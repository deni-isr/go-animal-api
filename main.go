package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Animal struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name" binding:"required,min=2,max=50" gorm:"size:50;not null"`
	Species    string    `json:"species" binding:"required,min=2,max=50" gorm:"size:50;not null"`
	Age        int       `json:"age" binding:"gte=0,lte=150" gorm:"not null"`
	Habitat    string    `json:"habitat" binding:"required,min=2,max=100" gorm:"size:100;not null"`
	Endangered bool      `json:"endangered" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var db *gorm.DB

func initDB() {
	_ = godotenv.Load()

	user := getEnv("DB_USER", "root")
	pass := getEnv("DB_PASS", "")
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "animal_db")

	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port)
	tempDB, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Не удалось подключиться к MariaDB (%s:%s): %v", host, port, err)
	}
	tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName))

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, dbName)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе %s: %v", dbName, err)
	}

	if err := db.AutoMigrate(&Animal{}); err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	var count int64
	db.Model(&Animal{}).Count(&count)
	if count == 0 {
		seedAnimals := []Animal{
			{Name: "Simba", Species: "Lion", Age: 5, Habitat: "Savannah", Endangered: true},
			{Name: "Luna", Species: "Cat", Age: 2, Habitat: "Domestic", Endangered: false},
			{Name: "Po", Species: "Giant Panda", Age: 7, Habitat: "Bamboo Forest", Endangered: true},
			{Name: "Rex", Species: "Dog", Age: 4, Habitat: "Domestic", Endangered: false},
			{Name: "Kowalski", Species: "Penguin", Age: 3, Habitat: "Antarctica", Endangered: false},
		}
		db.Create(&seedAnimals)
		log.Println("База данных заполнена тестовыми животными (5 записей)")
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// GET /api/v1/animals
func getAnimals(c *gin.Context) {
	query := db.Model(&Animal{})

	// 1. FILTERING
	if species := c.Query("species"); species != "" {
		query = query.Where("LOWER(species) = ?", strings.ToLower(species))
	}
	if habitat := c.Query("habitat"); habitat != "" {
		query = query.Where("LOWER(habitat) LIKE ?", "%"+strings.ToLower(habitat)+"%")
	}
	if endangered := c.Query("endangered"); endangered != "" {
		if isEndangered, err := strconv.ParseBool(endangered); err == nil {
			query = query.Where("endangered = ?", isEndangered)
		}
	}
	if minAge := c.Query("min_age"); minAge != "" {
		if age, err := strconv.Atoi(minAge); err == nil {
			query = query.Where("age >= ?", age)
		}
	}

	var total int64
	query.Count(&total)

	// 2. SORTING
	allowedSortFields := map[string]bool{
		"id":         true,
		"name":       true,
		"species":    true,
		"age":        true,
		"created_at": true,
	}
	sortBy := strings.ToLower(c.DefaultQuery("sort", "id"))
	if !allowedSortFields[sortBy] {
		sortBy = "id"
	}
	order := strings.ToUpper(c.DefaultQuery("order", "ASC"))
	if order != "ASC" && order != "DESC" {
		order = "ASC"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, order))

	// 3. PAGINATION
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var animals []Animal
	if err := query.Limit(limit).Offset(offset).Find(&animals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch animals"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"data": animals,
		"meta": gin.H{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GET /api/v1/animals/:id
func getAnimalByID(c *gin.Context) {
	var animal Animal
	if err := db.First(&animal, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
		return
	}
	c.JSON(http.StatusOK, animal)
}

// POST /api/v1/animals
func createAnimal(c *gin.Context) {
	var input Animal
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	if err := db.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create animal"})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// PUT /api/v1/animals/:id
func updateAnimal(c *gin.Context) {
	var animal Animal
	if err := db.First(&animal, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
		return
	}

	var input Animal
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	animal.Name = input.Name
	animal.Species = input.Species
	animal.Age = input.Age
	animal.Habitat = input.Habitat
	animal.Endangered = input.Endangered

	db.Save(&animal)
	c.JSON(http.StatusOK, animal)
}
func deleteAnimal(c *gin.Context) {
	var animal Animal
	if err := db.First(&animal, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Animal not found"})
		return
	}

	db.Delete(&animal)
	c.JSON(http.StatusOK, gin.H{"message": "Animal deleted successfully"})
}

func main() {
	initDB()

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Go Animal REST API",
			"docs":    "/swagger",
			"routes":  "/api/v1/animals",
		})
	})

	// Step 4: Отдача OpenAPI спецификации и Swagger UI
	r.StaticFile("/openapi.yaml", "./openapi.yaml")
	r.GET("/swagger", func(c *gin.Context) {
		html := `<!DOCTYPE html>
<html>
<head>
  <title>Animal API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/animals", getAnimals)
		v1.GET("/animals/:id", getAnimalByID)
		v1.POST("/animals", createAnimal)
		v1.PUT("/animals/:id", updateAnimal)
		v1.DELETE("/animals/:id", deleteAnimal)
	}

	port := getEnv("PORT", "8080")
	log.Printf("Сервер запущен на http://localhost:%s", port)
	log.Printf("Swagger документация: http://localhost:%s/swagger", port)
	r.Run(":" + port)
}