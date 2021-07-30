package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	RootPath string      `yaml:"rootPath"`
	MySQL    MySQLConfig `yaml:"mysql"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

var (
	configPath string
	config     Config
)

func init() {
	flag.StringVar(&configPath, "config", "config.yaml", "config path")
	flag.Parse()

	f, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		f.Close()
		log.Fatal(err.Error())
	}
}

func newDB(config *MySQLConfig) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)))
	if err != nil {
		return nil, nil, err
	}

	conn, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, conn, nil
}

func main() {
	db, conn, err := newDB(&config.MySQL)
	if err != nil {
		log.Fatal(err.Error())
	}
	db.AutoMigrate(&Count{})
	conn.Close()

	r := gin.Default()

	root := r.Group(config.RootPath)
	{
		root.GET("/api/count", countAPI)
	}

	if err := fcgi.Serve(nil, r); err != nil {
		log.Fatal(err.Error())
	}
}

type Count struct {
	gorm.Model
	Count int
}

func countAPI(ctx *gin.Context) {
	db, conn, err := newDB(&config.MySQL)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	defer conn.Close()

	count := Count{
		Model: gorm.Model{
			ID: 1,
		},
	}
	db.FirstOrCreate(&count)
	count.Count++
	db.Updates(&count)

	ctx.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}
