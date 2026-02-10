package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ngenohkevin/lms/internal/config"
	"github.com/ngenohkevin/lms/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Hash passwords
	adminPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash admin password:", err)
	}

	librarianPassword, err := bcrypt.GenerateFromPassword([]byte("librarian123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash librarian password:", err)
	}

	// Create admin user
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    updated_at = EXCLUDED.updated_at
	`, "admin", "admin@library.com", string(adminPassword), "admin", true, time.Now(), time.Now())

	if err != nil {
		log.Printf("Failed to create admin user: %v", err)
	} else {
		fmt.Println("✓ Admin user created/updated")
	}

	// Create librarian user
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO users (username, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    updated_at = EXCLUDED.updated_at
	`, "librarian", "librarian@library.com", string(librarianPassword), "librarian", true, time.Now(), time.Now())

	if err != nil {
		log.Printf("Failed to create librarian user: %v", err)
	} else {
		fmt.Println("✓ Librarian user created/updated")
	}

	// Create sample books
	books := []struct {
		BookID    string
		ISBN      string
		Title     string
		Author    string
		Publisher string
		Year      int
		Genre     string
		Copies    int
		Location  string
	}{
		{
			BookID:    "BK001",
			ISBN:      "978-0-7475-3269-6",
			Title:     "Harry Potter and the Philosopher's Stone",
			Author:    "J.K. Rowling",
			Publisher: "Bloomsbury",
			Year:      1997,
			Genre:     "Fantasy",
			Copies:    3,
			Location:  "Section A, Shelf 1",
		},
		{
			BookID:    "BK002",
			ISBN:      "978-0-06-112008-4",
			Title:     "To Kill a Mockingbird",
			Author:    "Harper Lee",
			Publisher: "J.B. Lippincott & Co.",
			Year:      1960,
			Genre:     "Fiction",
			Copies:    2,
			Location:  "Section B, Shelf 3",
		},
		{
			BookID:    "BK003",
			ISBN:      "978-0-14-044926-6",
			Title:     "1984",
			Author:    "George Orwell",
			Publisher: "Secker & Warburg",
			Year:      1949,
			Genre:     "Dystopian",
			Copies:    4,
			Location:  "Section C, Shelf 2",
		},
		{
			BookID:    "BK004",
			ISBN:      "978-0-316-76948-0",
			Title:     "The Catcher in the Rye",
			Author:    "J.D. Salinger",
			Publisher: "Little, Brown and Company",
			Year:      1951,
			Genre:     "Fiction",
			Copies:    2,
			Location:  "Section B, Shelf 4",
		},
		{
			BookID:    "BK005",
			ISBN:      "978-0-452-28423-4",
			Title:     "Brave New World",
			Author:    "Aldous Huxley",
			Publisher: "Chatto & Windus",
			Year:      1932,
			Genre:     "Dystopian",
			Copies:    3,
			Location:  "Section C, Shelf 3",
		},
	}

	for _, book := range books {
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO books (
				book_id, isbn, title, author, publisher,
				published_year, genre, total_copies, available_copies,
				shelf_location, is_active, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (book_id) DO UPDATE
			SET title = EXCLUDED.title,
			    author = EXCLUDED.author,
			    updated_at = EXCLUDED.updated_at
		`, book.BookID, book.ISBN, book.Title, book.Author, book.Publisher,
			book.Year, book.Genre, book.Copies, book.Copies,
			book.Location, true, time.Now(), time.Now())

		if err != nil {
			log.Printf("Failed to create book %s: %v", book.BookID, err)
		} else {
			fmt.Printf("✓ Book created/updated: %s\n", book.Title)
		}
	}

	// Create sample students
	students := []struct {
		StudentID  string
		FirstName  string
		LastName   string
		Email      string
		Phone      string
		Year       int
		Department string
	}{
		{
			StudentID:  "STU1",
			FirstName:  "John",
			LastName:   "Doe",
			Email:      "john.doe@school.edu",
			Phone:      "+1234567890",
			Year:       3,
			Department: "Computer Science",
		},
		{
			StudentID:  "STU2",
			FirstName:  "Jane",
			LastName:   "Smith",
			Email:      "jane.smith@school.edu",
			Phone:      "+1234567891",
			Year:       2,
			Department: "Literature",
		},
		{
			StudentID:  "STU3",
			FirstName:  "Robert",
			LastName:   "Johnson",
			Email:      "robert.johnson@school.edu",
			Phone:      "+1234567892",
			Year:       4,
			Department: "Engineering",
		},
	}

	for _, student := range students {
		// Hash a default password for students
		studentPassword, _ := bcrypt.GenerateFromPassword([]byte("student123"), bcrypt.DefaultCost)

		_, err = db.Pool.Exec(ctx, `
			INSERT INTO students (
				student_id, first_name, last_name, email, phone,
				year_of_study, department, password_hash, is_active,
				enrollment_date, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (student_id) DO UPDATE
			SET first_name = EXCLUDED.first_name,
			    last_name = EXCLUDED.last_name,
			    updated_at = EXCLUDED.updated_at
		`, student.StudentID, student.FirstName, student.LastName, student.Email,
			student.Phone, student.Year, student.Department, string(studentPassword),
			true, time.Now(), time.Now(), time.Now())

		if err != nil {
			log.Printf("Failed to create student %s: %v", student.StudentID, err)
		} else {
			fmt.Printf("✓ Student created/updated: %s %s\n", student.FirstName, student.LastName)
		}
	}

	fmt.Println("\n✅ Database seeding completed successfully!")
	fmt.Println("\nLogin Credentials:")
	fmt.Println("  Admin: username=admin, password=admin123")
	fmt.Println("  Librarian: username=librarian, password=librarian123")
	fmt.Println("  Students: password=student123")
}
