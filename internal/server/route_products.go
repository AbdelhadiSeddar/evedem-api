package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"evedem_api/internal/commons"
	"evedem_api/internal/database"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (c *Controller) register_products() {
	c.register_path("/v1/products", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/products", c.products_post)
	c.register_path("PUT /v1/products", c.products_put)

	c.Whitelist["/v1/products/img"] = true
	c.register_path("/v1/products/img", c.product_img_get)

	c.Whitelist["/v1/products/comments"] = true
	c.register_path("POST /v1/products/comments", c.products_comment_get)

  c.register_path("PUT /v1/products/order", c.products_order_put)
  c.register_path("POST /v1/products/order", c.products_order_post)
}
func (s *Controller) products_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Name        *string `json:"name,omitempty"`
		Height      *int    `json:"height,omitempty"`
		Width       *int    `json:"width,omitempty"`
		Depth       *int    `json:"depth,omitempty"`
		Quantity    *int    `json:"quantity,omitempty"`
		Price       *string `json:"price,omitempty"`
		CategoryId  *string `json:"categoryId,omitempty"` // Changed to string for flexibility
		Color       *string `json:"color,omitempty"`
		Description *string `json:"description,omitempty"`
		Picture     *string `json:"picture,omitempty"`
	}

	var b []byte
	{
		var err *commons.ApiError
		b, err = s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	req := BodyType{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	// Input Validation
	re := []string{}
	if req.Name == nil {
		re = append(re, "name")
	}
	if req.Height == nil {
		re = append(re, "height")
	}
	if req.Width == nil {
		re = append(re, "width")
	}
	if req.Depth == nil {
		re = append(re, "depth")
	}
	if req.Quantity == nil {
		re = append(re, "quantity")
	}
	if req.Price == nil {
		re = append(re, "price")
	}
	if req.CategoryId == nil {
		re = append(re, "categoryId")
	}
	if req.Color == nil {
		re = append(re, "color")
	}
	if req.Description == nil {
		re = append(re, "description")
	}
	if req.Picture == nil {
		re = append(re, "picture")
	}

	if len(re) > 0 {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: re,
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// Convert CategoryId to int (after validation)
	categoryIdInt := 0
	categoryId, err := strconv.Atoi(*req.CategoryId)
	if err != nil {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_INVALID, // Using existing error type
			Errorinfo: "Invalid CategoryId format.  Must be an integer string.",
			Data:      nil,
		}.HTTPSend(w)
		return
	}
	categoryIdInt = categoryId

	//Fetch the id
	auth, autherr := s.fetch_auth(r)
	if autherr != nil {
		autherr.HTTPSend(w)
		return
	}
	userid, usrerr := s.SessionGetUserUID(auth)
	if userid == nil || usrerr != nil {
		commons.ApiError{
			Error:     commons.ERR_AUTH_REQUIRED,
			Errorinfo: "User not authenticated", // or more detailed info as needed
		}.HTTPSend(w)
		return

	}

	// Database Connection
	db := database.GetDBConn()
	defer db.Release()

	// Construct the SQL query
	query := `
	INSERT INTO public."Product" (name, height, width, depth, quantity, price, "categoryId", color, description, picture, "sellerId", condition, reference)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	RETURNING "productId";
	`

	ref := *req.Name + time.Now().String()
	condition := "new"

	var productId int
	err = db.DB.QueryRow(context.Background(), query,
		*req.Name,
		*req.Height,
		*req.Width,
		*req.Depth,
		*req.Quantity,
		*req.Price,
		categoryIdInt, // Use the converted integer value
		*req.Color,
		*req.Description,
		*req.Picture,
		userid, // User the fetched userId
		condition,
		ref,
	).Scan(&productId)

	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)

		return
	}

	// Prepare the response data
	response := map[string]any{
		"productId": productId,
		"message":   "Product created successfully",
	}

	// Convert the response to JSON
	responseJson, err := json.Marshal(response)
	if err != nil {
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Unable to generate response body.",
		}.HTTPSend(w)
		return
	}

	// Set content type and send the response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(responseJson); err != nil {
		return
	}
}

func (s *Controller) products_put(w http.ResponseWriter, r *http.Request) {
	// Response struct
	type Response struct {
		ImageUrl string `json:"imageUrl"`
		Message  string `json:"message"`
	}

	// Request struct
	type Request struct {
		Image *string `json:"image"`
	}
	var b []byte
	{
		var err *commons.ApiError
		b, err = s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	req := Request{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}
	// Input Validation
	if req.Image == nil {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: "image is required",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// 3. Generate a UUID v4 filename
	newFilename := uuid.New().String()

	// Remove the data URL prefix (if present)
	dataURLPrefix := "data:image"
	base64Data := *req.Image
	if strings.Contains(base64Data, dataURLPrefix) {
		index := strings.Index(base64Data, ",")
		if index == -1 {
			log.Println("Invalid base64 data")
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: "Invalid base64 data",
				Data:      nil,
			}.HTTPSend(w)
			return
		}
		base64Data = base64Data[index+1:]
	}
	// Determine file extension from base64 header (very basic check, can be improved)
	var fileExtension string
	if strings.Contains(*req.Image, "data:image/jpeg;base64,") || strings.Contains(*req.Image, "data:image/jpg;base64,") {
		fileExtension = ".jpg"
	} else if strings.Contains(*req.Image, "data:image/png;base64,") {
		fileExtension = ".png"
	} else if strings.Contains(*req.Image, "data:image/gif;base64,") {
		fileExtension = ".gif"
	} else {
		log.Println("Error: Could not determine file extension.")
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_INVALID,
			Errorinfo: "Could not determine file extension.  Invalid file type.",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// Append the file extension to the UUID filename
	newFilenameWithExtension := newFilename + fileExtension

	// Decode base64 data
	decodedImage, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		log.Println("Error decoding base64 data:", err)
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_INVALID,
			Errorinfo: "Error decoding base64 data",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// Create folder if not exist
	err = os.MkdirAll("./uploads", os.ModeDir|0755)
	if err != nil {
		log.Println("Error creating directory:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Error creating directory",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// Uploads the file
	filePath := filepath.Join("./uploads", newFilenameWithExtension)
	file, err := os.Create(filePath)

	if err != nil {
		log.Println("Error creating file:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Error creating file",
			Data:      nil,
		}.HTTPSend(w)
		return
	}
	defer file.Close()

	// Write the decoded image data to the file
	_, err = file.Write(decodedImage)
	if err != nil {
		log.Println("Error writing image data to file:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Error writing image data to file",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	log.Println("Successfully Uploaded File")

	// 5. Return the URL of the uploaded image
	imageUrl := newFilenameWithExtension // Adjust based on your server configuration

	response := Response{
		ImageUrl: imageUrl,
		Message:  "Image uploaded successfully",
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Println("Error generating response body:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Unable to generate response body.",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(responseJson); err != nil {
		return
	}
}

// Helper function to determine the file extension
func getFileExtension(handler *multipart.FileHeader) string {
	name := handler.Filename
	segments := strings.Split(name, ".")
	if len(segments) > 1 {
		return "." + segments[len(segments)-1] // Return the last segment as the extension
	}
	return ""
}

func (s *Controller) product_img_get(w http.ResponseWriter, r *http.Request) {
	// 1. Get the filenameWithExtension from the query parameters
	keys, ok := r.URL.Query()["filename"]
	if !ok || len(keys[0]) < 1 {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: "Filename parameter is required",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	filename := keys[0]

	// Construct the full file path using the filename
	imageDirectory := "./uploads" // Adjust if images are stored elsewhere
	imagePath := filepath.Join(imageDirectory, filename)

	// Check if the file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		log.Printf("Image not found: %s", imagePath)
		http.NotFound(w, r) // Return a 404 Not Found
		return
	}

	// Open the file
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Error opening image: %s, error: %v", imagePath, err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Error opening the image file",
		}.HTTPSend(w)
		return
	}
	defer file.Close()

	// Set the Content-Type header based on the file extension
	contentType := getContentType(imagePath)
	w.Header().Set("Content-Type", contentType)

	// Copy the file content to the response
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("Error copying image to response: %s, error: %v", imagePath, err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Error serving the image file",
		}.HTTPSend(w)
		return
	}

	log.Printf("Served image: %s", imagePath)
}

func getContentType(imagePath string) string {
	ext := filepath.Ext(imagePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream" // Default to binary stream if type is unknown
	}
}
func (s *Controller) products_comment_get(w http.ResponseWriter, r *http.Request) {
	// 1. Extract CommentID from query parameters
	keys, ok := r.URL.Query()["productId"]
	if !ok || len(keys[0]) < 1 {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: "produtctId parameter is required",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	productIdStr := keys[0]

	// 2. Convert productId to integer
	productId, err := strconv.Atoi(productIdStr)
	if err != nil {
		log.Println("Invalid commentId format:", err)
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_INVALID,
			Errorinfo: "Invalid commentId format. Must be an integer string.",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// 3. Database Connection
	db := database.GetDBConn()
	defer db.Release()

	// 4. Construct the SQL query
	query := `
    SELECT jsonb_build_object(
        'commentId', c."commentId",
        'fullName', u."firstName" || ' ' || u."lastName",
        'productId', c."productId",
        'content', c.content,
        'date', c.date
    )
    FROM public."Comment" c
    JOIN public."User" u ON c."commenterId" = u."userId"
    WHERE c."productId" = $1
    ORDER BY c.date DESC;
`

	// 5. Query the database and scan directly into a byte slice
	var commentJSON []byte
	err = db.DB.QueryRow(context.Background(), query, productId).Scan(&commentJSON)

	if err != nil {
		log.Println("Error querying database:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		return
	}

	// 6. Set Content-Type header and send response
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(commentJSON); err != nil {
		return
	}
}
func (s *Controller) products_order_put(w http.ResponseWriter, r *http.Request) {
	type OrderItemRequest struct {
		ProductID *int `json:"productId,omitempty"`
		Quantity  *int `json:"quantity,omitempty"`
	}

	type BodyType struct {
		OrderItems []OrderItemRequest `json:"orderItems"`
	}

	var b []byte
	{
		var err *commons.ApiError
		b, err = s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	req := BodyType{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	// Input Validation (Order Items)
	if len(req.OrderItems) == 0 {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: []string{"orderItems (at least one)"},
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	// Further input validation for each order item.
	for i, item := range req.OrderItems {
		if item.ProductID == nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_MISSING,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].productId", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
		if item.Quantity == nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_MISSING,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].quantity", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
		if *item.Quantity <= 0 {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].quantity must be greater than 0", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	// Authentication
	auth, autherr := s.fetch_auth(r)
	if autherr != nil {
		autherr.HTTPSend(w)
		return
	}
	userid, usrerr := s.SessionGetUserUID(auth)
	if userid == nil || usrerr != nil {
		commons.ApiError{
			Error:     commons.ERR_AUTH_REQUIRED,
			Errorinfo: "User not authenticated", // or more detailed info as needed
		}.HTTPSend(w)
		return
	}

	// Database Connection
	db := database.GetDBConn()
	defer db.Release()

	// Begin a transaction for atomicity
	tx, err := db.DB.Begin(context.Background())
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(context.Background())
			panic(p) // re-throw panic after rollback
		} else if err != nil {
			tx.Rollback(context.Background())
		} else {
			err = tx.Commit(context.Background())
			if err != nil {
				log.Println("Failed to commit transaction:", err)
				// Log the error and send an appropriate response.
			}
		}
	}()
	// Insert into "Order" table
	orderDate := time.Now()
	var orderId int
	insertOrderQuery := `
		INSERT INTO public."Order" ("buyerId", status, date)
		VALUES ($1, $2, $3)
		RETURNING "orderId";
	`

	err = tx.QueryRow(context.Background(), insertOrderQuery,
		userid,          // Use the fetched userId for buyerId
		"pending",     // Set initial status
		orderDate,
	).Scan(&orderId)

	if err != nil {
		log.Println("Error inserting into Order:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		return
	}

	// Insert into "OrderItem" table for each item
	for _, item := range req.OrderItems {
		insertOrderItemQuery := `
			INSERT INTO public."OrderItem" ("orderId", "productId", quantity)
			VALUES ($1, $2, $3);
		`
		_, err = tx.Exec(context.Background(), insertOrderItemQuery,
			orderId,
			*item.ProductID,
			*item.Quantity,
		)
		if err != nil {
			log.Println("Error inserting into OrderItem:", err)
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}.HTTPSend(w)
			return // Rollback will happen due to the deferred function and the error.
		}
	}

	// Prepare the response (aggregate data)
	type OrderSummary struct {
		OrderID   int       `json:"orderId"`
		BuyerID   int       `json:"buyerId"`
		Status    string    `json:"status"`
		Date      time.Time `json:"date"`
		OrderItems []struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		} `json:"orderItems"`
	}

	orderSummary := OrderSummary{
		OrderID:  orderId,
		BuyerID:  *userid,  // Use userid here
		Status:   "pending", // Or fetch it from the order table if needed
		Date:     orderDate,
		OrderItems: make([]struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		}, len(req.OrderItems)),
	}

	for i, item := range req.OrderItems {
		orderSummary.OrderItems[i] = struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		}{
			ProductID: *item.ProductID,
			Quantity:  *item.Quantity,
		}
	}

	// Marshal the response to JSON
	responseJson, err := json.Marshal(orderSummary)
	if err != nil {
		log.Println("Error marshaling response:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Unable to generate response body.",
		}.HTTPSend(w)
		return
	}

	// Set content type and send the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Or http.StatusCreated
	if _, err := w.Write(responseJson); err != nil {
		log.Println("Error writing response:", err)
		//  Handle the error (e.g., log it).  The client might not receive anything.
	}
}

type User struct {
	UserID int `json:"userId"`
}

// Order model for the response
type Order struct {
	OrderID   int       `json:"orderId"`
	BuyerID   int       `json:"buyerId"`
	Status    string    `json:"status"`
	Date      time.Time `json:"date"`
	OrderItems []struct {
		ProductID int `json:"productId"`
		Quantity  int `json:"quantity"`
	} `json:"orderItems"`
}

// FetchOrdersResponse is a more complete response, including the Product details
type FetchOrdersResponse struct {
	Orders []Order `json:"orders"`
}

func (s *Controller) products_order_post(w http.ResponseWriter, r *http.Request) {
	type OrderItemRequest struct {
		ProductID *int `json:"productId,omitempty"`
		Quantity  *int `json:"quantity,omitempty"`
	}

	type BodyType struct {
		OrderItems []OrderItemRequest `json:"orderItems"`
	}

	type User struct {
		UserID   int    `json:"userId"`
		Username string `json:"username"` // Added username
	}

	type OrderItem struct {
		ProductID int `json:"productId"`
		Quantity  int `json:"quantity"`
	}

	type Order struct {
		OrderID    int         `json:"orderId"`
		BuyerID    int         `json:"buyerId"`
		BuyerName  string      `json:"buyerName"`
		Status     string      `json:"status"`
		Date       time.Time   `json:"date"`
		OrderItems []OrderItem `json:"orderItems"`
		IsSeller   bool        `json:"isSeller"`
	}

	type FetchOrdersResponse struct {
		Orders []Order `json:"orders"`
		UserID int     `json:"userId"`
	}

	var b []byte
	{
		var err *commons.ApiError
		b, err = s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	req := BodyType{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	if len(req.OrderItems) == 0 {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: []string{"orderItems (at least one)"},
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	for i, item := range req.OrderItems {
		if item.ProductID == nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_MISSING,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].productId", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
		if item.Quantity == nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_MISSING,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].quantity", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
		if *item.Quantity <= 0 {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: []string{fmt.Sprintf("orderItems[%d].quantity must be greater than 0", i)},
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	auth, autherr := s.fetch_auth(r)
	if autherr != nil {
		autherr.HTTPSend(w)
		return
	}
	userid, usrerr := s.SessionGetUserUID(auth)
	if userid == nil || usrerr != nil {
		commons.ApiError{
			Error:     commons.ERR_AUTH_REQUIRED,
			Errorinfo: "User not authenticated",
		}.HTTPSend(w)
		return
	}

	db := database.GetDBConn()
	defer db.Release()

	// Begin transaction
	tx, err := db.DB.Begin(context.Background())
	if err != nil {
		log.Println("Failed to begin transaction:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(context.Background())
			panic(p)
		} else if err != nil {
			tx.Rollback(context.Background())
		} else {
			err = tx.Commit(context.Background())
			if err != nil {
				log.Println("Failed to commit transaction:", err)
			}
		}
	}()

	orderDate := time.Now()
	var orderId int
	insertOrderQuery := `
		INSERT INTO public."Order" ("buyerId", status, date)
		VALUES ($1, $2, $3)
		RETURNING "orderId";
	`

	err = tx.QueryRow(context.Background(), insertOrderQuery,
		userid,
		"pending",
		orderDate,
	).Scan(&orderId)

	if err != nil {
		log.Println("Error inserting into Order:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		return
	}

	for _, item := range req.OrderItems {
		insertOrderItemQuery := `
			INSERT INTO public."OrderItem" ("orderId", "productId", quantity)
			VALUES ($1, $2, $3);
		`
		_, err = tx.Exec(context.Background(), insertOrderItemQuery,
			orderId,
			*item.ProductID,
			*item.Quantity,
		)
		if err != nil {
			log.Println("Error inserting into OrderItem:", err)
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}.HTTPSend(w)
			return
		}
	}

	// Prepare the response (aggregate data)
	type OrderSummary struct {
		OrderID   int       `json:"orderId"`
		BuyerID   int       `json:"buyerId"`
		Status    string    `json:"status"`
		Date      time.Time `json:"date"`
		OrderItems []struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		} `json:"orderItems"`
	}

	orderSummary := OrderSummary{
		OrderID:   orderId,
		BuyerID:   *userid,
		Status:    "pending",
		Date:      orderDate,
		OrderItems: make([]struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		}, len(req.OrderItems)),
	}

	for i, item := range req.OrderItems {
		orderSummary.OrderItems[i] = struct {
			ProductID int `json:"productId"`
			Quantity  int `json:"quantity"`
		}{
			ProductID: *item.ProductID,
			Quantity:  *item.Quantity,
		}
	}

	responseJson, err := json.Marshal(orderSummary)
	if err != nil {
		log.Println("Error marshaling response:", err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Unable to generate response body.",
		}.HTTPSend(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(responseJson); err != nil {
		log.Println("Error writing response:", err)
	}
}
