package controllers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/theleywin/Backend-Talent-Nest/src/lib"
	"github.com/theleywin/Backend-Talent-Nest/src/models"
	"gorm.io/gorm"
)

// GetFeedPosts returns posts for the authenticated user's feed, including posts from their connections and themselves
func GetFeedPosts(c *fiber.Ctx) error {
	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Obtener IDs de las conexiones aceptadas
	var connections []models.Connection
	err := lib.DB.Where("(sender_id = ? OR recipient_id = ?) AND status = ?",
		user.ID, user.ID, models.ConnectionStatusAccepted).
		Find(&connections).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching connections",
		})
	}

	// Crear array de IDs de usuarios conectados + el propio usuario
	connectionIDs := []uint{user.ID}
	for _, conn := range connections {
		if conn.SenderID == user.ID {
			connectionIDs = append(connectionIDs, conn.RecipientID)
		} else {
			connectionIDs = append(connectionIDs, conn.SenderID)
		}
	}

	// Buscar posts de usuarios conectados con Preload de relaciones
	var posts []models.Post
	err = lib.DB.Preload("Author").
		Preload("Likes").
		Preload("Comments.User").
		Preload("Repost.Author").
		Where("author_id IN ?", connectionIDs).
		Order("created_at DESC").
		Find(&posts).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching posts",
		})
	}

	// Convertir a PostDto
	var postDtos []models.PostDto
	for _, post := range posts {
		postDtos = append(postDtos, convertToPostDto(post))
	}

	return c.Status(fiber.StatusOK).JSON(postDtos)
}

// CreatePost creates a new post for the authenticated user, optionally uploading an image
func CreatePost(c *fiber.Ctx) error {
	type CreatePostRequest struct {
		Content string `json:"content"`
		Image   string `json:"image,omitempty"`  // Base64 string o URL
		Repost  *uint  `json:"repost,omitempty"` // ID del post a repostear
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

	// Procesar el campo Repost si existe
	var repostID *uint
	if req.Repost != nil && *req.Repost > 0 {
		// Verificar que el post a repostear existe
		var existingPost models.Post
		err := lib.DB.First(&existingPost, *req.Repost).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"message": "Post to repost not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Error verifying repost",
			})
		}

		repostID = req.Repost
	}

	// Crear nuevo post
	newPost := models.Post{
		AuthorID: user.ID,
		Content:  req.Content,
		Image:    imageURL,
		RepostID: repostID,
	}

	// Guardar en la base de datos
	if err := lib.DB.Create(&newPost).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to create post",
		})
	}

	// Cargar las relaciones para la respuesta
	lib.DB.Preload("Author").Preload("Repost.Author").First(&newPost, newPost.ID)

	return c.Status(fiber.StatusCreated).JSON(convertToPostDto(newPost))
}

// DeletePost deletes a post by ID if the authenticated user is the author
func DeletePost(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID",
		})
	}

	// Obtener usuario autenticado
	user := c.Locals("user").(models.User)

	// Buscar el post primero
	var post models.Post
	err = lib.DB.First(&post, uint(postID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post",
		})
	}

	// Verificar que el usuario es el autor del post
	if post.AuthorID != user.ID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "You are not authorized to delete this post",
		})
	}

	// Eliminar imagen de Cloudinary si existe
	if post.Image != "" {
		//TODO
		// err := deleteImageFromCloudinary(post.Image)
		// if err != nil {
		//     println("Error deleting image from Cloudinary:", err.Error())
		// }
	}

	// Eliminar comentarios y likes asociados (GORM lo hace automáticamente con OnDelete:CASCADE)
	// Eliminar el post de la base de datos
	if err := lib.DB.Delete(&post).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to delete post",
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
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	// Buscar el post por ID con todas las relaciones
	var post models.Post
	err = lib.DB.Preload("Author").
		Preload("Likes").
		Preload("Comments.User").
		Preload("Repost.Author").
		First(&post, uint(postID)).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post data",
		})
	}

	return c.Status(fiber.StatusOK).JSON(convertToPostDto(post))
}

// CreateComment adds a new comment to a post by its ID
func CreateComment(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	// Parsear el cuerpo de la solicitud
	type CreateCommentRequest struct {
		Content string `json:"content"`
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

	// Verificar que el post existe
	var post models.Post
	err = lib.DB.First(&post, uint(postID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post",
		})
	}

	// Crear el nuevo comentario
	newComment := models.Comment{
		PostID:  uint(postID),
		UserID:  user.ID,
		Content: req.Content,
	}

	if err := lib.DB.Create(&newComment).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to add comment",
		})
	}

	// Crear notificación si el comentarista no es el autor del post
	if post.AuthorID != user.ID {
		postIDUint := uint(postID)
		newNotification := models.Notification{
			RecipientID:   post.AuthorID,
			Type:          "comment",
			RelatedUserID: &user.ID,
			RelatedPostID: &postIDUint,
			Read:          false,
		}

		if err := lib.DB.Create(&newNotification).Error; err != nil {
			fmt.Printf("Error creating notification: %v\n", err)
		}
	}

	// Recargar el post con todas las relaciones
	err = lib.DB.Preload("Author").
		Preload("Likes").
		Preload("Comments.User").
		Preload("Repost.Author").
		First(&post, uint(postID)).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post details",
		})
	}

	return c.Status(fiber.StatusOK).JSON(convertToPostDto(post))
}

// LikePost toggles a like/unlike for a post by the authenticated user
func LikePost(c *fiber.Ctx) error {
	// Obtener ID del post desde los parámetros
	postIDStr := c.Params("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid post ID format",
		})
	}

	// Obtener usuario autenticado del middleware
	user := c.Locals("user").(models.User)

	// Buscar el post
	var post models.Post
	err = lib.DB.Preload("Likes").First(&post, uint(postID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Post not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error fetching post",
		})
	}

	// Verificar si el usuario ya dio like al post
	var existingLike models.Like
	err = lib.DB.Where("post_id = ? AND user_id = ?", uint(postID), user.ID).First(&existingLike).Error

	var shouldCreateNotification bool

	if err == nil {
		// Ya existe el like, eliminarlo (unlike)
		if err := lib.DB.Delete(&existingLike).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to unlike post",
			})
		}
		shouldCreateNotification = false
	} else if err == gorm.ErrRecordNotFound {
		// No existe el like, crearlo
		newLike := models.Like{
			PostID: uint(postID),
			UserID: user.ID,
		}
		if err := lib.DB.Create(&newLike).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to like post",
			})
		}
		// Crear notificación solo si el usuario no es el autor del post
		shouldCreateNotification = (post.AuthorID != user.ID)
	} else {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error checking like status",
		})
	}

	// Crear notificación si es necesario
	if shouldCreateNotification {
		postIDUint := uint(postID)
		newNotification := models.Notification{
			RecipientID:   post.AuthorID,
			Type:          "like",
			RelatedUserID: &user.ID,
			RelatedPostID: &postIDUint,
			Read:          false,
		}

		if err := lib.DB.Create(&newNotification).Error; err != nil {
			fmt.Printf("Error creating notification: %v\n", err)
		}
	}

	// Recargar el post con todas las relaciones
	err = lib.DB.Preload("Author").
		Preload("Likes").
		Preload("Comments.User").
		Preload("Repost.Author").
		First(&post, uint(postID)).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error loading post details",
		})
	}

	return c.Status(fiber.StatusOK).JSON(convertToPostDto(post))
}

// Helper function to convert Post model to PostDto
func convertToPostDto(post models.Post) models.PostDto {
	postDto := models.PostDto{
		ID: post.ID,
		Author: models.UserDto{
			ID:             post.Author.ID,
			Name:           post.Author.Name,
			Username:       post.Author.Username,
			ProfilePicture: post.Author.ProfilePicture,
			Headline:       post.Author.HeadLine,
		},
		Content:   post.Content,
		Image:     post.Image,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}

	// Convert Likes
	for _, like := range post.Likes {
		postDto.Likes = append(postDto.Likes, models.UserDto{
			ID:             like.User.ID,
			Name:           like.User.Name,
			Username:       like.User.Username,
			ProfilePicture: like.User.ProfilePicture,
			Headline:       like.User.HeadLine,
		})
	}

	// Convert Comments
	for _, comment := range post.Comments {
		postDto.Comments = append(postDto.Comments, models.CommentDto{
			ID:      comment.ID,
			Content: comment.Content,
			User: models.UserDto{
				ID:             comment.User.ID,
				Name:           comment.User.Name,
				Username:       comment.User.Username,
				ProfilePicture: comment.User.ProfilePicture,
				Headline:       comment.User.HeadLine,
			},
			CreatedAt: comment.CreatedAt,
		})
	}

	// Convert Repost if exists
	if post.RepostID != nil && post.Repost != nil {
		repostDto := models.PostDto{
			ID: post.Repost.ID,
			Author: models.UserDto{
				ID:             post.Repost.Author.ID,
				Name:           post.Repost.Author.Name,
				Username:       post.Repost.Author.Username,
				ProfilePicture: post.Repost.Author.ProfilePicture,
				Headline:       post.Repost.Author.HeadLine,
			},
			Content:   post.Repost.Content,
			Image:     post.Repost.Image,
			CreatedAt: post.Repost.CreatedAt,
			UpdatedAt: post.Repost.UpdatedAt,
		}
		postDto.Repost = &repostDto
	}

	return postDto
}
