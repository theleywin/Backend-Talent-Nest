package cluster

import (
	"encoding/json"
	"log"
	"time"

	"gorm.io/gorm"
)

// ReplicationHook es un plugin de GORM que captura operaciones de escritura
type ReplicationHook struct {
	ClusterState *ClusterState
}

// Name retorna el nombre del plugin
func (h *ReplicationHook) Name() string {
	return "ReplicationHook"
}

// Initialize inicializa el plugin y registra los callbacks
func (h *ReplicationHook) Initialize(db *gorm.DB) error {
	// Callback DESPUÉS de INSERT
	db.Callback().Create().After("gorm:create").Register("replication:after_create", h.afterCreate)

	// Callback DESPUÉS de UPDATE
	db.Callback().Update().After("gorm:update").Register("replication:after_update", h.afterUpdate)

	// Callback DESPUÉS de DELETE
	db.Callback().Delete().After("gorm:delete").Register("replication:after_delete", h.afterDelete)

	log.Println("✓ Replication hooks registered in GORM")
	return nil
}

// afterCreate se ejecuta después de cada INSERT
func (h *ReplicationHook) afterCreate(db *gorm.DB) {
	// Solo replicar si somos el líder
	if !h.ClusterState.IsLeader() {
		return
	}

	// Verificar que no hubo error
	if db.Error != nil {
		return
	}

	// Verificar que tenemos el statement y schema
	statement := db.Statement
	if statement == nil || statement.Schema == nil {
		return
	}

	tableName := statement.Schema.Table

	// Extraer todos los campos del modelo insertado
	data := make(map[string]interface{})

	for _, field := range statement.Schema.Fields {
		if field.Readable && field.DBName != "" {
			var value interface{}
			var isZero bool

			// Para campos con serializador JSON, obtener el valor real del struct
			if field.Serializer != nil {
				fieldValue := statement.ReflectValue.FieldByName(field.Name)
				if fieldValue.IsValid() && fieldValue.CanInterface() {
					value = fieldValue.Interface()
					isZero = fieldValue.IsZero()
				} else {
					continue
				}
			} else {
				value, isZero = field.ValueOf(db.Statement.Context, statement.ReflectValue)
			}

			if !isZero {
				// Solo agregar valores primitivos o serializables
				cleanValue := sanitizeValue(value)
				if cleanValue != nil {
					data[field.DBName] = cleanValue
				}
			}
		}
	}

	// Obtener el ID del registro insertado
	var recordID uint
	if idField := statement.Schema.PrioritizedPrimaryField; idField != nil {
		value, _ := idField.ValueOf(db.Statement.Context, statement.ReflectValue)
		if id, ok := value.(uint); ok {
			recordID = id
		}
	}

	log.Printf("[ReplicationHook] 📝 Captured INSERT on %s (ID=%d): %v", tableName, recordID, data)

	// Replicar de forma asíncrona
	go func() {
		h.ClusterState.ReplicateToFollowers("INSERT", tableName, data, recordID)
		log.Printf("[ReplicationHook] ✓ Replicated INSERT on %s (ID=%d) to followers", tableName, recordID)
	}()
}

// afterUpdate se ejecuta después de cada UPDATE
func (h *ReplicationHook) afterUpdate(db *gorm.DB) {
	// Solo replicar si somos el líder
	if !h.ClusterState.IsLeader() {
		return
	}

	// Verificar que no hubo error
	if db.Error != nil {
		return
	}

	statement := db.Statement
	if statement == nil || statement.Schema == nil {
		return
	}

	tableName := statement.Schema.Table

	// Extraer datos actualizados
	data := make(map[string]interface{})

	// Si se usó Updates() con un map, los datos están en statement.Dest
	if destMap, ok := statement.Dest.(map[string]interface{}); ok {
		// Usar directamente los campos del map que se pasaron a Updates()
		for k, v := range destMap {
			cleanValue := sanitizeValue(v)
			if cleanValue != nil {
				data[k] = cleanValue
			}
		}
		log.Printf("[ReplicationHook] UPDATE using map with %d fields", len(data))
	} else {
		// Si se usó Save() o Updates() con struct, extraer TODOS los campos
		for _, field := range statement.Schema.Fields {
			if field.Readable && field.DBName != "" && field.DBName != "id" {
				// Para campos con serializador JSON, obtener el valor real del struct
				var value interface{}
				if field.Serializer != nil {
					// Campo serializado: obtener el valor directamente del struct
					fieldValue := statement.ReflectValue.FieldByName(field.Name)
					if fieldValue.IsValid() && fieldValue.CanInterface() {
						value = fieldValue.Interface()
						log.Printf("[ReplicationHook] Field %s (serialized, type: %T)", field.DBName, value)
					} else {
						log.Printf("[ReplicationHook] ⚠️ Cannot access serialized field %s", field.DBName)
						continue
					}
				} else {
					// Campo normal: usar ValueOf
					value, _ = field.ValueOf(db.Statement.Context, statement.ReflectValue)
					log.Printf("[ReplicationHook] Field %s (type: %T)", field.DBName, value)
				}

				// No filtrar por isZero porque Save() guarda todo
				cleanValue := sanitizeValue(value)
				if cleanValue != nil {
					data[field.DBName] = cleanValue
					log.Printf("[ReplicationHook] ✓ Added field %s to replication data", field.DBName)
				} else {
					log.Printf("[ReplicationHook] ⚠️ Field %s returned nil after sanitization (type: %T)", field.DBName, value)
				}
			}
		}
		log.Printf("[ReplicationHook] UPDATE using struct with %d fields", len(data))
	}

	// Siempre incluir updated_at
	data["updated_at"] = time.Now().Format(time.RFC3339)

	// Obtener el ID del registro actualizado
	var recordID uint
	if idField := statement.Schema.PrioritizedPrimaryField; idField != nil {
		value, _ := idField.ValueOf(db.Statement.Context, statement.ReflectValue)
		if id, ok := value.(uint); ok {
			recordID = id
		}
	}

	log.Printf("[ReplicationHook] 📝 Captured UPDATE on %s (ID=%d): %v", tableName, recordID, data)

	// Replicar de forma asíncrona
	go func() {
		h.ClusterState.ReplicateToFollowers("UPDATE", tableName, data, recordID)
		log.Printf("[ReplicationHook] ✓ Replicated UPDATE on %s (ID=%d) to followers", tableName, recordID)
	}()
}

// afterDelete se ejecuta después de cada DELETE
func (h *ReplicationHook) afterDelete(db *gorm.DB) {
	// Solo replicar si somos el líder
	if !h.ClusterState.IsLeader() {
		return
	}

	// Verificar que no hubo error
	if db.Error != nil {
		return
	}

	statement := db.Statement
	if statement == nil || statement.Schema == nil {
		return
	}

	tableName := statement.Schema.Table

	// Obtener el ID del registro eliminado
	var recordID uint
	if idField := statement.Schema.PrioritizedPrimaryField; idField != nil {
		value, _ := idField.ValueOf(db.Statement.Context, statement.ReflectValue)
		if id, ok := value.(uint); ok {
			recordID = id
		}
	}

	// Para soft delete, enviar el timestamp
	data := map[string]interface{}{
		"deleted_at": time.Now(),
	}

	log.Printf("[ReplicationHook] 📝 Captured DELETE on %s (ID=%d)", tableName, recordID)

	// Replicar de forma asíncrona
	go func() {
		h.ClusterState.ReplicateToFollowers("DELETE", tableName, data, recordID)
		log.Printf("[ReplicationHook] ✓ Replicated DELETE on %s (ID=%d) to followers", tableName, recordID)
	}()
}

// sanitizeValue convierte valores complejos a tipos serializables en JSON
func sanitizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// Tipos primitivos se devuelven directamente
		return v
	case time.Time:
		// Convertir time.Time a string ISO
		return v.Format(time.RFC3339)
	case []byte:
		// Bytes pueden ser JSON serializado por GORM, intentar deserializar
		if len(v) == 0 {
			return nil
		}
		var jsonData interface{}
		if err := json.Unmarshal(v, &jsonData); err == nil {
			log.Printf("[sanitizeValue] ✓ Deserialized []byte to JSON: %v", jsonData)
			return jsonData
		}
		// Si no es JSON válido, devolver como string
		return string(v)
	case []string:
		return v
	case []int:
		return v
	case []uint:
		return v
	case []float64:
		return v
	case []interface{}:
		// Slice de interfaces (pueden ser arrays JSON)
		log.Printf("[sanitizeValue] ✓ Found []interface{} with %d elements", len(v))
		return v
	case []map[string]interface{}:
		// Slice de mapas (experience, education, etc.)
		log.Printf("[sanitizeValue] ✓ Found []map[string]interface{} with %d elements", len(v))
		return v
	case map[string]interface{}:
		// Map genérico
		return v
	default:
		// Para otros tipos complejos, intentar serializar como JSON
		// Esto captura structs, slices de structs, etc.
		log.Printf("[sanitizeValue] Attempting to serialize type %T", v)
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			log.Printf("[sanitizeValue] ❌ Cannot serialize value of type %T: %v", v, err)
			return nil
		}

		// Deserializar de nuevo para obtener un tipo nativo de Go
		var jsonData interface{}
		if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
			log.Printf("[sanitizeValue] ❌ Cannot deserialize JSON: %v", err)
			return nil
		}

		log.Printf("[sanitizeValue] ✓ Serialized type %T to: %v", v, jsonData)
		return jsonData
	}
}
