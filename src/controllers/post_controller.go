package controllers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetFeedPosts returns posts for the authenticated user's feed, including posts from their connections and themselves
func GetFeedPosts(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Crear array de IDs para la consulta (connections + propio usuario)
	connectionIDs := make([]primitive.ObjectID, len(user.Connections))
	copy(connectionIDs, user.Connections)
	connectionIDs = append(connectionIDs, user.Id)

	collection := lib.DB.Collection("posts")

	// Consulta equivalente a: Post.find({ author: { $in: connections } })
	filter := bson.M{
		"author": bson.M{
			"$in": connectionIDs,
		},
	}

	opts := options.Find().SetSort(bson.M{"createdAt": -1})
	cursor, err := collection.Find(c.Context(), filter, opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching posts",
		})
	}
	defer cursor.Close(c.Context())

	var posts []models.Post
	if err := cursor.All(c.Context(), &posts); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error decoding posts",
		})
	}

	// Populate manual de autores y comentarios
	populatedPosts, err := lib.PopulatePosts(c, posts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error populating posts",
		})
	}

	return c.Status(fiber.StatusOK).JSON(populatedPosts)
}

// CreatePost creates a new post for the authenticated user, optionally uploading an image
func CreatePost(c *fiber.Ctx) error {
	type CreatePostRequest struct {
		Content string `json:"content"`
		Image   string `json:"image,omitempty"` // Base64 string o URL
	}

	var req CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	var imageURL string

	// Si hay imagen, subir a Cloudinary
	if req.Image != "" {
		//TODO
		// Configurar Cloudinary (deberías tener esto en un package aparte)
		// cld, err := cloudinary.NewFromURL("cloudinary://api_key:api_secret@cloud_name")
		// if err != nil {
		//     return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		//         "message": "Error configuring Cloudinary",
		//     })
		// }

		// // Subir imagen a Cloudinary
		// uploadResult, err := cld.Upload.Upload(context.Background(), req.Image, uploader.UploadParams{})
		// if err != nil {
		//     return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		//         "message": "Error uploading image to Cloudinary",
		//     })
		// }

		// imageURL = uploadResult.SecureURL
	}

	// Crear nuevo post
	newPost := models.Post{
		Id:        primitive.NewObjectID(),
		Author:    user.Id,
		Content:   req.Content,
		Image:     imageURL,
		Likes:     []primitive.ObjectID{},
		Comments:  []models.Comment{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Guardar en MongoDB
	collection := lib.DB.Collection("posts")
	_, err := collection.InsertOne(c.Context(), newPost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create post",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(newPost)
}

// DeletePost deletes a post by ID if the authenticated user is the author
func DeletePost(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID",
		})
	}

	// Obtener usuario autenticado
	user := c.Locals("user").(models.User)

	collection := lib.DB.Collection("posts")

	// Buscar el post primero
	var post models.Post
	err = collection.FindOne(c.Context(), bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Post not found",
		})
	}

	// Verificar que el usuario es el autor del post
	if post.Author != user.Id {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "You are not authorized to delete this post",
		})
	}

	// Eliminar imagen de Cloudinary si existe
	if post.Image != "" {
		//TODO
		// err := deleteImageFromCloudinary(post.Image)
		// if err != nil {
		//     // Puedes decidir si continuar con la eliminación del post aunque falle Cloudinary
		//     // o retornar error. Aquí continúa pero loguea el error.
		//     println("Error deleting image from Cloudinary:", err.Error())
		// }
	}

	// Eliminar el post de la base de datos
	result, err := collection.DeleteOne(c.Context(), bson.M{"_id": postID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to delete post",
		})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Post not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Post deleted successfully",
	})
}

// GetPostByID returns a post by its ID, including populated author and comments
func GetPostByID(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	collection := lib.DB.Collection("posts")

	// Buscar el post por ID
	var post models.Post
	err = collection.FindOne(c.Context(), bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Post not found",
		})
	}

	// Popular manualmente los datos del autor y comentarios
	populatedPost, err := lib.PopulatePosts(c, []models.Post{post})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post data",
		})
	}

	return c.Status(fiber.StatusOK).JSON(populatedPost[0])
}

// CreateComment adds a new comment to a post by its ID
func CreateComment(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	// Parsear el cuerpo de la solicitud
	type CreateCommentRequest struct {
		Content string `json:"content" bson:"content"`
	}

	var req CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	// Validar que el contenido no esté vacío
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Comment content cannot be empty",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Crear el nuevo comentario
	newComment := models.Comment{
		Id:        primitive.NewObjectID(),
		User:      user.Id,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	// Configurar la actualización
	update := bson.M{
		"$push": bson.M{
			"comments": newComment,
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	// Opciones para devolver el documento actualizado
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	// Ejecutar la actualización (equivalente a findByIdAndUpdate con {new: true})
	postsCollection := lib.DB.Collection("posts")
	var updatedPost models.Post
	err = postsCollection.FindOneAndUpdate(
		c.Context(),
		bson.M{"_id": postID},
		update,
		opts,
	).Decode(&updatedPost)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to add comment",
		})
	}

	// Popular el autor del post
	usersCollection := lib.DB.Collection("users")
	var postAuthor models.User
	projection := bson.M{
		"name":           1,
		"email":          1,
		"username":       1,
		"headline":       1,
		"profilePicture": 1,
	}

	err = usersCollection.FindOne(
		c.Context(),
		bson.M{"_id": updatedPost.Author},
		options.FindOne().SetProjection(projection),
	).Decode(&postAuthor)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post author",
		})
	}

	// Crear notificación si el comentarista no es el autor del post
	//TODO Implement Notifications first
	// if postAuthor.Id != user.Id {
	// 	newNotification := models.Notification{
	// 		ID:          primitive.NewObjectID(),
	// 		Recipient:   postAuthor.Id,
	// 		Type:        "comment",
	// 		RelatedUser: user.Id,
	// 		RelatedPost: postID,
	// 		CreatedAt:   time.Now(),
	// 		IsRead:      false,
	// 	}

	// 	notificationsCollection := database.DB.Collection("notifications")
	// 	_, err = notificationsCollection.InsertOne(c.Context(), newNotification)
	// 	if err != nil {
	// 		// Log del error pero continuar (la notificación no es crítica)
	// 		println("Error creating notification:", err.Error())
	// 	}
	// }

	populatedPost, err := lib.PopulatePosts(c, []models.Post{updatedPost})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post details",
		})
	}

	return c.Status(fiber.StatusOK).JSON(populatedPost[0])
}

// LikePost toggles a like/unlike for a post by the authenticated user
func LikePost(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := primitive.ObjectIDFromHex(postIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	postsCollection := lib.DB.Collection("posts")

	// Primero buscar el post para verificar si ya tiene like
	var post models.Post
	err = postsCollection.FindOne(c.Context(), bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post",
		})
	}

	// Verificar si el usuario ya dio like al post
	alreadyLiked := false
	for _, likeID := range post.Likes {
		if likeID == user.Id {
			alreadyLiked = true
			break
		}
	}

	var update bson.M
	//TODOvar shouldCreateNotification bool

	if alreadyLiked {
		// Quitar like (unlike)
		update = bson.M{
			"$pull": bson.M{"likes": user.Id},
			"$set":  bson.M{"updatedAt": time.Now()},
		}
		//TODOshouldCreateNotification = false
	} else {
		// Agregar like
		update = bson.M{
			"$push": bson.M{"likes": user.Id},
			"$set":  bson.M{"updatedAt": time.Now()},
		}
		// Crear notificación solo si el usuario no es el autor del post
		//TODOshouldCreateNotification = (post.Author != user.Id)
	}

	// Actualizar el post
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	var updatedPost models.Post
	err = postsCollection.FindOneAndUpdate(
		c.Context(),
		bson.M{"_id": postID},
		update,
		opts,
	).Decode(&updatedPost)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update post",
		})
	}

	// Crear notificación si es necesario
	// TODO implement notifications first
	// if shouldCreateNotification {
	// 	newNotification := models.Notification{
	// 		ID:          primitive.NewObjectID(),
	// 		Recipient:   post.Author,
	// 		Type:        "like",
	// 		RelatedUser: user.ID,
	// 		RelatedPost: postID,
	// 		CreatedAt:   time.Now(),
	// 		IsRead:      false,
	// 	}

	// 	notificationsCollection := database.DB.Collection("notifications")
	// 	_, err = notificationsCollection.InsertOne(c.Context(), newNotification)
	// 	if err != nil {
	// 		// Log del error pero continuar (la notificación no es crítica)
	// 		println("Error creating notification:", err.Error())
	// 	}
	// }

	// Popular el post actualizado para la respuesta
	populatedPost, err := lib.PopulatePosts(c, []models.Post{updatedPost})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post details",
		})
	}

	return c.Status(fiber.StatusOK).JSON(populatedPost[0])
}
