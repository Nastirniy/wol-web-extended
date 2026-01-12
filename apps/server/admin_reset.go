package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// resetAdminPassword performs an interactive password reset for superusers.
// This function uses database transactions to ensure atomic operations - if the
// process is interrupted at any point, the transaction will automatically roll back,
// leaving the original password intact.
//
// Process flow:
//  1. Query all superusers from database
//  2. Display numbered list for user selection
//  3. Prompt for new password (hidden input)
//  4. Prompt for password confirmation (hidden input)
//  5. Begin transaction
//  6. Update password in database
//  7. Commit transaction
//
// If interrupted at any step before commit, the transaction rolls back automatically.
// This prevents the corner case where a user could end up without a valid password.
func resetAdminPassword(db *sql.DB) error {
	fmt.Printf("=================================================\n")
	fmt.Printf("Admin Password Reset\n")
	fmt.Printf("=================================================\n")

	// Step 1: Get all superusers
	superusers, err := getSuperusers(db)
	if err != nil {
		return fmt.Errorf("failed to query superusers: %w", err)
	}

	if len(superusers) == 0 {
		fmt.Printf("No superusers found in database.\n")
		fmt.Printf("Start the server normally and visit /auth to create a superuser.\n")
		return nil
	}

	// Step 2: Display superusers and prompt for selection
	fmt.Printf("\nFound %d superuser(s):\n", len(superusers))
	for i, user := range superusers {
		fmt.Printf("  [%d] %s (ID: %s, Created: %s)\n", i+1, user.Name, user.ID, user.Created.Format("2006-01-02 15:04:05"))
	}
	fmt.Printf("\n")

	selectedUser, err := promptUserSelection(superusers)
	if err != nil {
		return fmt.Errorf("user selection failed: %w", err)
	}

	fmt.Printf("\nSelected user: %s\n", selectedUser.Name)
	fmt.Printf("\n")

	// Step 3: Prompt for new password
	newPassword, err := promptPassword("Enter new password: ")
	if err != nil {
		return fmt.Errorf("password input failed: %w", err)
	}

	// Step 4: Prompt for password confirmation
	confirmPassword, err := promptPassword("Confirm new password: ")
	if err != nil {
		return fmt.Errorf("password confirmation failed: %w", err)
	}

	// Validate passwords match
	if newPassword != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Validate password is not empty
	if len(strings.TrimSpace(newPassword)) == 0 {
		return fmt.Errorf("password cannot be empty")
	}

	// Step 5: Begin transaction for atomic update
	fmt.Printf("\nUpdating password for user '%s'...\n", selectedUser.Name)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on any error
	defer func() {
		if err != nil {
			tx.Rollback()
			fmt.Printf("Transaction rolled back due to error\n")
		}
	}()

	// Step 6: Hash password and update in database
	hashedPassword := hashPassword(newPassword)

	result, err := tx.Exec("UPDATE users SET password = ?, updated = CURRENT_TIMESTAMP WHERE id = ?",
		hashedPassword, selectedUser.ID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify update: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated - user may have been deleted")
	}

	// Step 7: Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("SUCCESS: Password updated for user '%s'\n", selectedUser.Name)
	fmt.Printf("You can now login with the new password.\n")
	fmt.Printf("=================================================\n")

	return nil
}

// getSuperusers retrieves all superusers from the database
func getSuperusers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`
		SELECT id, name, password, readonly, is_superuser, created, updated
		FROM users
		WHERE is_superuser = TRUE
		ORDER BY created ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Name, &user.Password, &user.ReadOnly, &user.IsSuperuser, &user.Created, &user.Updated)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// promptUserSelection prompts the user to select a superuser from the list
func promptUserSelection(users []User) (*User, error) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("Select user (1-%d): ", len(users))

		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		selection, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number between 1 and %d.\n", len(users))
			continue
		}

		if selection < 1 || selection > len(users) {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d.\n", len(users))
			continue
		}

		return &users[selection-1], nil
	}
}

// promptPassword prompts for a password with hidden input
// Returns the password string or an error if reading failed
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	// Read password without echoing
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Print newline after hidden input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(passwordBytes), nil
}
