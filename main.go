package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "INFO: ", log.LstdFlags)

	// Retrieve environment variables
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	action := os.Getenv("ACTION")

	// Check if required environment variables are set
	if mysqlHost == "" || mysqlPort == "" || mysqlUser == "" || mysqlPassword == "" {
		logger.Println("One or more environment variables are missing")
		os.Exit(1)
	}

	if action == "" {
		logger.Println("ACTION environment variable is missing")
		os.Exit(1)
	}

	// Perform the action
	switch action {
	case "backup":
		performBackup(logger, mysqlHost, mysqlPort, mysqlUser, mysqlPassword)
	case "restore":
		performRestore(logger, mysqlHost, mysqlPort, mysqlUser, mysqlPassword)
	default:
		logger.Printf("Invalid ACTION: %s\n", action)
		os.Exit(1)
	}
}

func performBackup(logger *log.Logger, host, port, user, password string) {
	logger.Println("Performing database backup...")

	// Execute mysqldump command
	cmd := exec.Command("mysqldump", "-h", host, "-P", port, "-u", user, "-p"+password, "--all-databases")
	output, err := cmd.CombinedOutput()

	// Log and handle errors
	if err != nil {
		logger.Printf("Backup failed: %v\n", err)
		logger.Println(string(output))
		os.Exit(1)
	}

	// Write output to backup file
	backupFilePath := "/backup/all_databases_backup.sql"
	err = os.WriteFile(backupFilePath, output, 0644)
	if err != nil {
		logger.Printf("Failed to write backup file: %v\n", err)
		os.Exit(1)
	}

	logger.Println("Backup successful")
}

func performRestore(logger *log.Logger, host, port, user, password string) {
	logger.Println("Performing database restore...")

	// Read backup file
	backupFilePath := "/backup/all_databases_backup.sql"
	input, err := os.ReadFile(backupFilePath)
	if err != nil {
		logger.Printf("Failed to read backup file: %v\n", err)
		os.Exit(1)
	}

	// Execute mysql command
	cmd := exec.Command("mysql", "-h", host, "-P", port, "-u", user, "-p"+password)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	// Log and handle errors
	if err != nil {
		logger.Printf("Restore failed: %v\n", err)
		os.Exit(1)
	}

	logger.Println("Restore successful")
}
