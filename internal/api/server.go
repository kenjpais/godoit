package api

import (
	"doit/internal/api/middlewares"
	"doit/internal/controller"
	"doit/internal/db"
	"doit/internal/cache/redishandler"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

var (
	RedisClient = redishandler.GetRedisClient()
)

func StartServer() {
	r := gin.Default()

	r.Use(middlewares.RateLimitMiddleware)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/jobs", listJobs)
		v1.POST("/job", createJob)
		v1.POST("/job-script", uploadJob)
		v1.GET("/job/:id", getJob)
		v1.PUT("/job", updateJob)
		v1.DELETE("/job/:id", deleteJob)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// listJobs retrieves jobs with pagination from the database.
func listJobs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var jobs []db.Job
	if err := db.DB.Limit(limit).Offset(offset).Find(&jobs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}

	// Add HATEOAS links
	response := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		response[i] = map[string]interface{}{
			"job": job,
			"_links": map[string]string{
				"self": fmt.Sprintf("/jobs/%s", job.JobID),
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   response,
		"limit":  limit,
		"offset": offset,
	})
}

func createJob(c *gin.Context) {
	var job db.Job

	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	jc, err := controller.NewJobController("JobOperationController")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Fatal(err.Error())
		return
	}

	if err := jc.CreateJob(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Job created successfully", "job": job})
}

func uploadJob(c *gin.Context) {
	file, err := c.FormFile("script")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed"})
		return
	}
	if file.Header.Get("Content-Type") != "text/x-python" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid MIME type, expected Python script"})
		return
	}
	jobId := c.Request.Header.Get("job_id")
	jc, err := controller.NewJobController("JobOperationController")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Fatal(err.Error())
		return
	}
	if err := jc.UploadJob(file, jobId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Job uploaded successfully"})
}


func getJob(c *gin.Context) {
	jobID := c.Param("id")

	jc, err := controller.NewJobController("JobOperationController")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Fatal(err.Error())
		return
	}

	job, err := jc.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

func updateJob(c *gin.Context) {
	var job db.Job

	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	jc, err := controller.NewJobController("JobOperationController")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Fatal(err.Error())
		return
	}

	if err := jc.UpdateJob(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to update job in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job updated successfully", "job": job})
}

func deleteJob(c *gin.Context) {
	jobID := c.Param("id")

	jc, err := controller.NewJobController("JobOperationController")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		log.Fatal(err.Error())
		return
	}

	if err := jc.DeleteJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to delete job: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}
