package cluster

import (
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
			value, isZero := field.ValueOf(db.Statement.Context, statement.ReflectValue)
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
				value, _ := field.ValueOf(db.Statement.Context, statement.ReflectValue)
				// No filtrar por isZero porque Save() guarda todo
				cleanValue := sanitizeValue(value)
				if cleanValue != nil {
					data[field.DBName] = cleanValue
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
		// Convertir bytes a string
		return string(v)
	case []string, []int, []uint, []float64:
		// Slices de tipos primitivos son seguros
		return v
	default:
		// Ignorar tipos complejos que no se pueden serializar
		return nil
	}
}
