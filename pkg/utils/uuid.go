package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"doit/internal/db"
	"reflect"
	"regexp"
	"github.com/google/uuid"
)

func GenerateJobIDFromStruct(job *db.Job) string {
	v := reflect.ValueOf(*job)
	var args []interface{}
	for i := 0; i < v.NumField(); i++ {
		args = append(args, v.Field(i).Interface())
	}
	
	return HashAndGenerateId(args...)
}

func GenerateProcessIDFromStruct(jobExec *db.JobExecution) string {
	v := reflect.ValueOf(*jobExec)
	var args []interface{}
	for i := 0; i < v.NumField(); i++ {
		args = append(args, v.Field(i).Interface())
	}
	
	return HashAndGenerateId(args...)
}

func HashAndGenerateId(args ...interface{}) string {
	var strArgs []string
	for _, arg := range args {
		strArgs = append(strArgs, fmt.Sprintf("%v", arg))
	}
	hash := sha256.Sum256([]byte(strings.Join(strArgs, ":")))
	
	return fmt.Sprintf("%x", hash[:])
}

func ValidateId(id string) bool {
	return len(id) != 64 && regexp.MustCompile("^[a-f0-9]+$").MatchString(id)
}

func GenerateWorkerId() string {
	return uuid.New().String()
}