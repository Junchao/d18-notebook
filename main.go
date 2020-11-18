package main

import (
	"database/sql"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/speed18/d18-notebook/log"
	"github.com/speed18/d18-notebook/server"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"strings"
)

func printDelimiter() {
	fmt.Println("----------------------------------")
}

func initLogger() (*os.File, error) {
	var logFile *os.File
	log.Logger = logrus.New()

	toStdout := viper.Sub("log").GetBool("stdout")
	if !toStdout {
		filename := os.Getenv("NOTEBOOK_LOG")
		if filename == "" {
			fmt.Println("load log file from env failed, use log_dev.log instead")
			filename = "log/log_dev.log"
		} else {
			filename = "log/" + filename
			fmt.Printf("log file: %s\n", filename)
		}
		logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		log.Logger.Out = logFile
	} else {
		fmt.Println("will log to stdout")
	}

	log.Logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logLevel := strings.ToLower(viper.GetString("log_level"))
	if logLevel == "" {
		fmt.Println("no log level configured, use default level")
		logLevel = "info"
		log.Logger.SetLevel(logrus.InfoLevel)
	} else if logLevel == "debug" {
		log.Logger.SetLevel(logrus.DebugLevel)
	} else if logLevel == "info" {
		log.Logger.SetLevel(logrus.InfoLevel)
	} else {
		fmt.Printf("Unsupported level: %s, use default level\n", logLevel)
		logLevel = "info"
		log.Logger.SetLevel(logrus.InfoLevel)
	}
	fmt.Printf("log level: %s\n", logLevel)

	fmt.Println("logger inited")
	return logFile, nil
}

func initConfig() error {
	configName := os.Getenv("NOTEBOOK_CONFIG")
	if configName == "" {
		fmt.Println("load config file from env failed, use config_dev.yaml instead")
		configName = "config_dev"
	} else {
		fmt.Printf("config file: %s\n", configName)
	}
	viper.SetConfigName(configName)
	viper.AddConfigPath("./config")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Printf("config file has changed, reload now\n")
	})
	fmt.Println("config inited")

	return nil
}

func initMysql() error {
	mysqlConfig := viper.Sub("mysql")
	var mysqlConfigStr strings.Builder
	mysqlConfigStr.WriteString(mysqlConfig.GetString("user"))
	mysqlConfigStr.WriteString(":")
	mysqlConfigStr.WriteString(mysqlConfig.GetString("password"))
	mysqlConfigStr.WriteString("@tcp(")
	mysqlConfigStr.WriteString(mysqlConfig.GetString("addr"))
	mysqlConfigStr.WriteString(":")
	mysqlConfigStr.WriteString(mysqlConfig.GetString("port"))
	mysqlConfigStr.WriteString(")/")
	dsn := mysqlConfigStr.String()

	var err error
	server.DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	fmt.Printf("mysql inited\n")

	return nil
}

func initRedis() error {
	redisConfig := viper.Sub("redis")
	server.RDS = redis.NewClient(
		&redis.Options{Addr: redisConfig.GetString("addr") + ":" + redisConfig.GetString("port"),
			DB: redisConfig.GetInt("DB")})
	fmt.Println("redis inited")
	return nil
}

func main() {
	if err := initConfig(); err != nil {
		fmt.Printf("init config failed: %s\n", err.Error())
		return
	}
	printDelimiter()

	if err := initMysql(); err != nil {
		fmt.Printf("init mysql failed: %s\n", err.Error())
		return
	}
	printDelimiter()

	if err := initRedis(); err != nil {
		fmt.Printf("init redis failed: %s\n", err.Error())
		return
	}
	printDelimiter()

	logFile, err := initLogger()
	if err != nil {
		fmt.Printf("init logger failed: %s\n", err.Error())
		return
	}
	defer func() {
		if logFile == nil {
			fmt.Println("logFile is nil, perhaps log to stdout?")
			return
		}
		err := logFile.Close()
		if err != nil {
			fmt.Printf("ERROR! close log file failed: %s\n", err.Error())
		} else {
			fmt.Println("log file closed")
		}
	}()

	http.HandleFunc("/api/note/publish", server.NotePublishHandler)
	http.HandleFunc("/api/notes", server.NotesHandler)
	http.HandleFunc("/api/note", server.NoteHandler)
	http.HandleFunc("/api/note/update", server.NoteUpdateHandler)
	http.HandleFunc("/api/note/delete", server.NoteDeleteHandler)
	http.HandleFunc("/api/tags", server.TagsHandler)
	http.HandleFunc("/api/auth", server.AuthHandler)
	http.HandleFunc("/api/is_auth", server.IsAuthHandler)
	http.HandleFunc("/api/logout", server.LogoutHandler)

	printDelimiter()

	port := viper.GetString("server.port")
	fmt.Printf("all set up! notebook server listening on :" + port + "...\n")
	err = http.ListenAndServe(":"+port, nil)
	log.Logger.Panic(err)
}
