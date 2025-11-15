package router

import (
	"net/http"
	"reviewer_pr/api"
	httpapi "reviewer_pr/internal/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Router(h *httpapi.Handler) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	r.GET("/openapi.yml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-yaml", api.OpenAPISpec)
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/openapi.yml"),
	))

	r.POST("/team/add", h.TeamAdd)
	r.GET("/team/get", h.TeamGet)

	r.POST("/users/setIsActive", h.UserSetIsActive)
	r.GET("/users/getReview", h.UserGetReview)

	r.POST("/pullRequest/create", h.PRCreate)
	r.POST("/pullRequest/merge", h.PRMerge)
	r.POST("/pullRequest/reassign", h.PRReassign)

	r.GET("/stats", h.GetStats)

	return r
}
