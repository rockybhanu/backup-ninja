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

	// Print all environment variables for troubleshooting
	logger.Println("Printing all environment variables:")
	for _, e := range os.Environ() {
		logger.Println(e)
	}

	// Retrieve environment variables
	envVars := []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"RESTIC_REPOSITORY",
		"RESTIC_PASSWORD",
		"ACTION",
		"RESTIC_HOSTNAME",
		"BACKUP_MOUNT_PATH",
		"RESTORE_MOUNT_PATH",
		"BACKUP_TYPE",
	}

	for _, v := range envVars {
		if os.Getenv(v) == "" {
			logger.Printf("Environment variable %s is not set", v)
			os.Exit(1)
		}
	}

	backupType := os.Getenv("BACKUP_TYPE")
	action := os.Getenv("ACTION")

	logger.Printf("Environment Variables: BACKUP_TYPE=%s, ACTION=%s",
		backupType, action)

	// Perform the action
	switch action {
	case "backup":
		initializeRepository(logger)
		logSnapshots(logger)
		if backupType == "db" {
			performDBBackup(logger)
		} else if backupType == "pvc" {
			performPVCBackup(logger)
		} else {
			logger.Printf("Invalid BACKUP_TYPE: %s\n", backupType)
			os.Exit(1)
		}
	case "restore":
		logSnapshots(logger)
		if backupType == "db" {
			performDBRestore(logger)
		} else if backupType == "pvc" {
			performPVCRestore(logger)
		} else {
			logger.Printf("Invalid BACKUP_TYPE: %s\n", backupType)
			os.Exit(1)
		}
	default:
		logger.Printf("Invalid ACTION: %s\n", action)
		os.Exit(1)
	}
}

func initializeRepository(logger *log.Logger) {
	logger.Println("Initializing Restic repository if necessary...")
	cmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "snapshots")
	if err := cmd.Run(); err != nil {
		logger.Println("Restic repository does not exist. Initializing...")
		initCmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "init")
		if err := initCmd.Run(); err != nil {
			logger.Printf("Failed to initialize Restic repository: %v\n", err)
			os.Exit(1)
		}
		logger.Println("Restic repository initialized successfully.")
	} else {
		logger.Println("Restic repository found.")
	}
}

func logSnapshots(logger *log.Logger) {
	logger.Printf("Listing all snapshots for host %s:", os.Getenv("RESTIC_HOSTNAME"))
	cmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "snapshots")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Printf("Failed to list snapshots: %v\n", err)
		logger.Println(string(output))
		os.Exit(1)
	}
	logger.Println(string(output))
}

func performDBBackup(logger *log.Logger) {
	logger.Println("Performing database backup...")

	// Retrieve database environment variables
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")

	logger.Printf("Environment Variables: MYSQL_HOST=%s, MYSQL_PORT=%s, MYSQL_USER=%s, MYSQL_PASSWORD=%s",
		mysqlHost, mysqlPort, mysqlUser, mysqlPassword)

	// Execute mysqldump command
	cmd := exec.Command("mysqldump", "-h", mysqlHost, "-P", mysqlPort, "-u", mysqlUser, "-p"+mysqlPassword, "--all-databases")
	output, err := cmd.CombinedOutput()

	// Log and handle errors
	if err != nil {
		logger.Printf("Backup failed: %v\n", err)
		logger.Println(string(output))
		os.Exit(1)
	}

	// Check if mysqldump output is empty
	if len(output) == 0 {
		logger.Println("Backup failed: mysqldump output is empty")
		os.Exit(1)
	}

	// Write output to backup file
	backupFilePath := "/backup/all_databases_backup.sql"
	err = os.WriteFile(backupFilePath, output, 0644)
	if err != nil {
		logger.Printf("Failed to write backup file: %v\n", err)
		os.Exit(1)
	}

	logger.Println("Database backup completed successfully.")

	logger.Printf("Starting backup of %s to %s with host %s", os.Getenv("BACKUP_MOUNT_PATH"), os.Getenv("RESTIC_REPOSITORY"), os.Getenv("RESTIC_HOSTNAME"))
	backupCmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "backup", os.Getenv("BACKUP_MOUNT_PATH"))
	backupOutput, err := backupCmd.CombinedOutput()
	if err != nil {
		logger.Printf("Restic backup failed: %v\n", err)
		logger.Println(string(backupOutput))
		os.Exit(1)
	}
	logger.Println("Backup completed successfully.")
	logger.Println(string(backupOutput))
	logSnapshots(logger)
	os.Exit(0)
}

func performDBRestore(logger *log.Logger) {
	logger.Println("Performing database restore...")

	// Ensure the latest snapshot is used for restoration unless specified otherwise
	snapshotID := os.Getenv("SNAPSHOT_ID")
	if snapshotID == "" {
		snapshotID = "latest"
	}

	logger.Printf("Starting restore of %s to %s with host %s", snapshotID, os.Getenv("RESTORE_MOUNT_PATH"), os.Getenv("RESTIC_HOSTNAME"))
	restoreCmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "restore", snapshotID, "--target", os.Getenv("RESTORE_MOUNT_PATH"))
	restoreOutput, err := restoreCmd.CombinedOutput()
	if err != nil {
		logger.Printf("Restic restore failed: %v\n", err)
		logger.Println(string(restoreOutput))
		os.Exit(1)
	}
	logger.Println("Restore completed successfully.")
	logger.Println(string(restoreOutput))

	// Read backup file
	backupFilePath := "/backup/all_databases_backup.sql"
	input, err := os.ReadFile(backupFilePath)
	if err != nil {
		logger.Printf("Failed to read backup file: %v\n", err)
		os.Exit(1)
	}

	// Check if backup file is empty
	if len(input) == 0 {
		logger.Println("Restore failed: backup file is empty")
		os.Exit(1)
	}

	logger.Println("Backup file validated successfully.")

	// Retrieve database environment variables
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")

	logger.Printf("Environment Variables: MYSQL_HOST=%s, MYSQL_PORT=%s, MYSQL_USER=%s, MYSQL_PASSWORD=%s",
		mysqlHost, mysqlPort, mysqlUser, mysqlPassword)

	// Execute mysql command
	cmd := exec.Command("mysql", "-h", mysqlHost, "-P", mysqlPort, "-u", mysqlUser, "-p"+mysqlPassword)
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
	os.Exit(0)
}

func performPVCBackup(logger *log.Logger) {
	logger.Println("Starting PVC backup...")

	logger.Printf("Starting backup of %s to %s with host %s", os.Getenv("BACKUP_MOUNT_PATH"), os.Getenv("RESTIC_REPOSITORY"), os.Getenv("RESTIC_HOSTNAME"))
	backupCmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "backup", os.Getenv("BACKUP_MOUNT_PATH"))
	backupOutput, err := backupCmd.CombinedOutput()
	if err != nil {
		logger.Printf("Restic backup failed: %v\n", err)
		logger.Println(string(backupOutput))
		os.Exit(1)
	}
	logger.Println("Backup completed successfully.")
	logger.Println(string(backupOutput))
	logSnapshots(logger)
	os.Exit(0)
}

func performPVCRestore(logger *log.Logger) {
	logger.Println("Starting PVC restore...")

	// Ensure the latest snapshot is used for restoration unless specified otherwise
	snapshotID := os.Getenv("SNAPSHOT_ID")
	if snapshotID == "" {
		snapshotID = "latest"
	}

	logger.Printf("Starting restore of %s to %s with host %s", snapshotID, os.Getenv("RESTORE_MOUNT_PATH"), os.Getenv("RESTIC_HOSTNAME"))
	restoreCmd := exec.Command("restic", "-r", os.Getenv("RESTIC_REPOSITORY"), "--host", os.Getenv("RESTIC_HOSTNAME"), "restore", snapshotID, "--target", os.Getenv("RESTORE_MOUNT_PATH"))
	restoreOutput, err := restoreCmd.CombinedOutput()
	if err != nil {
		logger.Printf("Restic restore failed: %v\n", err)
		logger.Println(string(restoreOutput))
		os.Exit(1)
	}
	logger.Println("Restore completed successfully.")
	logger.Println(string(restoreOutput))
	logSnapshots(logger)
	os.Exit(0)
}
