package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jung-kurt/gofpdf"
	"gopkg.in/gomail.v2"
)

type Product struct {
	ID          int
	Title       string
	Description string
	Price       float64
	Quantity    int
	Active      bool
}

type Client struct {
	ID        int
	FirstName string
	LastName  string
	Phone     string
	Address   string
	Email     string
}

type Order struct {
	ID        int
	ClientID  int
	ProductID int
	Quantity  int
	Price     float64
	OrderDate time.Time
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/")
	if err != nil {
		panic(err)
	}

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS shop"); err != nil {
		panic(err)
	}

	if _, err := db.Exec("USE shop"); err != nil {
		panic(err)
	}

	createProductsTable := `
	CREATE TABLE IF NOT EXISTS products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(100) NOT NULL,
		description TEXT,
		price FLOAT NOT NULL,
		quantity INT NOT NULL,
		active BOOLEAN DEFAULT TRUE
	)`

	createClientsTable := `
	CREATE TABLE IF NOT EXISTS clients (
		id INT AUTO_INCREMENT PRIMARY KEY,
		first_name VARCHAR(50) NOT NULL,
		last_name VARCHAR(50) NOT NULL,
		phone VARCHAR(20),
		address VARCHAR(255),
		email VARCHAR(100) NOT NULL
	)`

	createOrdersTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id INT AUTO_INCREMENT PRIMARY KEY,
		client_id INT NOT NULL,
		product_id INT NOT NULL,
		quantity INT NOT NULL,
		price FLOAT NOT NULL,
		order_date DATETIME NOT NULL,
		FOREIGN KEY (client_id) REFERENCES clients(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	)`

	_, err = db.Exec(createProductsTable)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(createClientsTable)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(createOrdersTable)
	if err != nil {
		panic(err)
	}

	color.Green("Database and tables created successfully.")
}

func addProduct() {
	scanner := bufio.NewScanner(os.Stdin)

	var quantity int
	var title, description string
	var price float64

	for {
		color.White("Enter product title:")
		if scanner.Scan() {
			title = scanner.Text()
			title = strings.TrimSpace(title)
		}
		if title != "" {
			break
		}
		color.Red("Title cannot be empty. Please enter a valid title.")
	}

	for {
		color.White("Enter product description:")
		if scanner.Scan() {
			description = scanner.Text()
			description = strings.TrimSpace(description)
		}
		if description != "" {
			break
		}
		color.Red("Description cannot be empty. Please enter a valid description.")
	}

	for {
		color.White("Enter product price:")
		if scanner.Scan() {
			priceStr := scanner.Text()
			priceTemp, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				color.Red("Invalid price. Please enter a valid number.")
				continue
			}
			price = round(priceTemp, 2)
		}
		break
	}

	for {
		color.White("Enter product quantity:")
		if scanner.Scan() {
			quantityStr := scanner.Text()
			qty, err := strconv.Atoi(quantityStr)
			if err != nil {
				color.Red("Invalid quantity. Please enter a valid integer.")
				continue
			}
			quantity = qty
		}
		break
	}

	_, err := db.Exec("INSERT INTO products (title, description, price, quantity, active) VALUES (?, ?, ?, ?, ?)",
		title, description, price, quantity, true)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	color.Green("Product added successfully.")
}

func round(num float64, decimalPlaces int) float64 {
	pow := math.Pow(10, float64(decimalPlaces))
	return math.Round(num*pow) / pow
}

func displayProducts() {
	color.White("List of Products:")

	rows, err := db.Query("SELECT id, title, description, price, quantity FROM products WHERE active = 1")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprintln(w, "ID\tTitle\tDescription\tPrice\tQuantity")
	fmt.Fprintln(w, "--\t-----\t-----------\t-----\t--------")

	for rows.Next() {
		var product Product
		err := rows.Scan(&product.ID, &product.Title, &product.Description, &product.Price, &product.Quantity)
		if err != nil {
			color.Red("Error:", err)
			return
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%.2f\t%d\n",
			product.ID, product.Title, product.Description, product.Price, product.Quantity)
	}

	w.Flush()
}

func modifyProduct() {
	var id int
	color.White("Enter product ID to modify:")
	fmt.Scan(&id)

	var title, description string
	var price float64
	var quantity int

	color.White("Enter new title:")
	fmt.Scan(&title)
	color.White("Enter new description:")
	fmt.Scan(&description)
	color.White("Enter new price:")
	fmt.Scan(&price)
	color.White("Enter new quantity:")
	fmt.Scan(&quantity)

	_, err := db.Exec("UPDATE products SET title = ?, description = ?, price = ?, quantity = ? WHERE id = ?",
		title, description, price, quantity, id)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	color.Green("Product modified successfully.")
}

func deleteProduct() {
	var id int
	color.White("Enter product ID to deactivate:")
	fmt.Scan(&id)

	_, err := db.Exec("UPDATE products SET active = 0 WHERE id = ?", id)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	color.Green("Product deactivated successfully.")
}

func exportProductsToCSV() {
	file, err := os.Create("products.csv")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"ID", "Title", "Description", "Price", "Quantity"}
	writer.Write(header)

	rows, err := db.Query("SELECT id, title, description, price, quantity FROM products")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var product Product
		err := rows.Scan(&product.ID, &product.Title, &product.Description, &product.Price, &product.Quantity)
		if err != nil {
			color.Red("Error:", err)
			return
		}

		record := []string{
			fmt.Sprint(product.ID),
			product.Title,
			product.Description,
			fmt.Sprint(product.Price),
			fmt.Sprint(product.Quantity),
		}
		writer.Write(record)
	}

	color.Green("Products exported to CSV successfully.")
}

func addClient() {
	scanner := bufio.NewScanner(os.Stdin)

	var firstName, lastName, phone, address, email string

	for {
		color.White("Enter client first name:")
		if scanner.Scan() {
			firstName = strings.TrimSpace(scanner.Text())
		}
		if firstName != "" {
			break
		}
		color.Red("First name cannot be empty. Please enter a valid first name.")
	}

	for {
		color.White("Enter client last name:")
		if scanner.Scan() {
			lastName = strings.TrimSpace(scanner.Text())
		}
		if lastName != "" {
			break
		}
		color.Red("Last name cannot be empty. Please enter a valid last name.")
	}

	for {
		color.White("Enter client phone number:")
		if scanner.Scan() {
			phone = strings.TrimSpace(scanner.Text())
		}
		if isValidPhone(phone) {
			break
		}
		color.Red("Phone number must contain only digits. Please enter a valid phone number.")
	}

	for {
		color.White("Enter client address:")
		if scanner.Scan() {
			address = strings.TrimSpace(scanner.Text())
		}
		if address != "" {
			break
		}
		color.Red("Address cannot be empty. Please enter a valid address.")
	}

	for {
		color.White("Enter client email:")
		if scanner.Scan() {
			email = strings.TrimSpace(scanner.Text())
		}

		if email != "" && isValidEmail(email) {
			break
		} else {
			color.Red("Invalid email format or empty email. Please try again.")
		}
	}

	_, err := db.Exec("INSERT INTO clients (first_name, last_name, phone, address, email) VALUES (?, ?, ?, ?, ?)",
		firstName, lastName, phone, address, email)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	color.Green("Client added successfully.")
}

func isValidPhone(phone string) bool {
	const phoneRegex = `^\d+$`
	re := regexp.MustCompile(phoneRegex)
	return re.MatchString(phone)
}

func isValidEmail(email string) bool {
	const emailRegex = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

func displayClients() {
	color.White("List of Clients:")

	rows, err := db.Query("SELECT id, first_name, last_name, phone, address, email FROM clients")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprintln(w, "ID\tFirst Name\tLast Name\tPhone\tAddress\tEmail")
	fmt.Fprintln(w, "--\t----------\t---------\t-----\t-------\t-----")

	for rows.Next() {
		var client Client
		err := rows.Scan(&client.ID, &client.FirstName, &client.LastName, &client.Phone, &client.Address, &client.Email)
		if err != nil {
			color.Red("Error:", err)
			return
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			client.ID, client.FirstName, client.LastName, client.Phone, client.Address, client.Email)
	}

	w.Flush()
}

func modifyClient() {
	var id int
	color.White("Enter client ID to modify:")
	fmt.Scan(&id)

	var firstName, lastName, phone, address, email string
	color.White("Enter new first name:")
	fmt.Scan(&firstName)
	color.White("Enter new last name:")
	fmt.Scan(&lastName)
	color.White("Enter new phone number:")
	fmt.Scan(&phone)
	color.White("Enter new address:")
	fmt.Scan(&address)
	color.White("Enter new email:")
	fmt.Scan(&email)

	_, err := db.Exec("UPDATE clients SET first_name = ?, last_name = ?, phone = ?, address = ?, email = ? WHERE id = ?",
		firstName, lastName, phone, address, email, id)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	color.Green("Client modified successfully.")
}

func exportClientsToCSV() {
	file, err := os.Create("clients.csv")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"ID", "First Name", "Last Name", "Phone", "Address", "Email"}
	writer.Write(header)

	rows, err := db.Query("SELECT id, first_name, last_name, phone, address, email FROM clients")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var client Client
		err := rows.Scan(&client.ID, &client.FirstName, &client.LastName, &client.Phone, &client.Address, &client.Email)
		if err != nil {
			color.Red("Error:", err)
			return
		}

		record := []string{
			fmt.Sprint(client.ID),
			client.FirstName,
			client.LastName,
			client.Phone,
			client.Address,
			client.Email,
		}
		writer.Write(record)
	}

	color.Green("Clients exported to CSV successfully.")
}

func makeOrder() {
	scanner := bufio.NewScanner(os.Stdin)

	var clientID, productID, quantity int

	for {
		fmt.Print("Enter client ID:")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "" {
				color.Red("Client ID cannot be empty. Please enter a valid ID.")
				continue
			}
			id, err := strconv.Atoi(input)
			if err != nil {
				color.Red("Invalid client ID. Please enter a valid integer.")
				continue
			}
			clientID = id
		}
		break
	}

	for {
		fmt.Print("Enter product ID:")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "" {
				color.Red("Product ID cannot be empty. Please enter a valid ID.")
				continue
			}
			id, err := strconv.Atoi(input)
			if err != nil {
				color.Red("Invalid product ID. Please enter a valid integer.")
				continue
			}
			productID = id
		}
		break
	}

	for {
		fmt.Print("Enter quantity:")
		if scanner.Scan() {
			input := scanner.Text()
			if input == "" {
				color.Red("Quantity cannot be empty. Please enter a valid quantity.")
				continue
			}
			qty, err := strconv.Atoi(input)
			if err != nil {
				color.Red("Invalid quantity. Please enter a valid integer.")
				continue
			}
			quantity = qty
		}
		break
	}

	var clientExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM clients WHERE id = ?)", clientID).Scan(&clientExists)
	if err != nil {
		color.Red("Error:", err)
		return
	}
	if !clientExists {
		color.Red("Client not found.")
		return
	}

	var productExists bool
	var productPrice float64
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = ?)", productID).Scan(&productExists)
	if err != nil {
		color.Red("Error:", err)
		return
	}
	if !productExists {
		color.Red("Product not found.")
		return
	}

	err = db.QueryRow("SELECT price FROM products WHERE id = ?", productID).Scan(&productPrice)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	orderDate := time.Now()
	result, err := db.Exec("INSERT INTO orders (client_id, product_id, quantity, price, order_date) VALUES (?, ?, ?, ?, ?)",
		clientID, productID, quantity, productPrice*float64(quantity), orderDate)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	orderID, err := result.LastInsertId()
	if err != nil {
		color.Red("Error:", err)
		return
	}

	order := Order{
		ID:        int(orderID),
		ClientID:  clientID,
		ProductID: productID,
		Quantity:  quantity,
		Price:     productPrice * float64(quantity),
		OrderDate: orderDate,
	}

	pdf := generateOrderPDF(order)
	sendOrderConfirmationEmail(clientID, order, pdf)

	color.Green("Order placed successfully.")
}

func sendOrderConfirmationEmail(clientID int, order Order, pdf *gofpdf.Fpdf) {
	var client Client
	err := db.QueryRow("SELECT first_name, email FROM clients WHERE id = ?", clientID).Scan(&client.FirstName, &client.Email)
	if err != nil {
		color.Red("Error:", err)
		return
	}

	tmpFile, err := os.CreateTemp("", "order_confirmation_*.pdf")
	if err != nil {
		color.Red("Could not create temporary file:", err)
		return
	}
	defer tmpFile.Close()

	err = pdf.OutputFileAndClose(tmpFile.Name())
	if err != nil {
		color.Red("Could not save PDF to temporary file:", err)
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "no-reply@yourdomain.com")
	m.SetHeader("To", client.Email)
	m.SetHeader("Subject", "Order Confirmation")
	m.SetBody("text/plain", fmt.Sprintf("Dear %s,\n\nThank you for your order.\nOrder ID: %d\nProduct ID: %d\nQuantity: %d\nTotal Price: %.2f\n\nBest regards,\nYour Company",
		client.FirstName, order.ID, order.ProductID, order.Quantity, order.Price))

	m.Attach(tmpFile.Name())

	d := gomail.NewDialer("sandbox.smtp.mailtrap.io", 587, "c22e6da0a15ac4", "499db904a3db25")
	d.TLSConfig = nil

	if err := d.DialAndSend(m); err != nil {
		color.Red("Could not send email:", err)
		return
	}

	color.Green("Order confirmation email sent successfully.")
}

func generateOrderPDF(order Order) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	pdf.Cell(40, 10, "Order Confirmation")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Order ID: %d", order.ID))
	pdf.Ln(10)
	pdf.Cell(40, 10, fmt.Sprintf("Client ID: %d", order.ClientID))
	pdf.Ln(10)
	pdf.Cell(40, 10, fmt.Sprintf("Product ID: %d", order.ProductID))
	pdf.Ln(10)
	pdf.Cell(40, 10, fmt.Sprintf("Quantity: %d", order.Quantity))
	pdf.Ln(10)
	pdf.Cell(40, 10, fmt.Sprintf("Total Price: %.2f", order.Price))

	err := pdf.OutputFileAndClose("order.pdf")
	if err != nil {
		color.Red("Could not generate PDF:", err)
	}

	color.Green("Order PDF generated successfully.")

	return pdf
}

func exportOrdersToCSV() {
	file, err := os.Create("orders.csv")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	rows, err := db.Query("SELECT id, client_id, product_id, quantity, price, order_date FROM orders")
	if err != nil {
		color.Red("Error:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		var orderDate string
		err := rows.Scan(&order.ID, &order.ClientID, &order.ProductID, &order.Quantity, &order.Price, &orderDate)
		if err != nil {
			color.Red("Error:", err)
			return
		}

		record := []string{
			fmt.Sprint(order.ID),
			fmt.Sprint(order.ClientID),
			fmt.Sprint(order.ProductID),
			fmt.Sprint(order.Quantity),
			fmt.Sprint(order.Price),
			orderDate,
		}
		writer.Write(record)
	}

	color.Green("Orders exported to CSV successfully.")
}

func main() {
	initDB()
	defer db.Close()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		color.White("Menu:")
		color.White("1. Add Product")
		color.White("2. Display Products")
		color.White("3. Modify Product")
		color.White("4. Deactivate Product")
		color.White("5. Export Products to CSV")
		color.White("6. Add Client")
		color.White("7. Display Clients")
		color.White("8. Modify Client")
		color.White("9. Export Clients to CSV")
		color.White("10. Make an Order")
		color.White("11. Export Orders to CSV")
		color.White("12. Exit")
		fmt.Print("Choose an option: ")

		var choice int
		if scanner.Scan() {
			choice, _ = strconv.Atoi(scanner.Text())
		}

		switch choice {
		case 1:
			addProduct()
		case 2:
			displayProducts()
		case 3:
			modifyProduct()
		case 4:
			deleteProduct()
		case 5:
			exportProductsToCSV()
		case 6:
			addClient()
		case 7:
			displayClients()
		case 8:
			modifyClient()
		case 9:
			exportClientsToCSV()
		case 10:
			makeOrder()
		case 11:
			exportOrdersToCSV()
		case 12:
			color.Cyan("Exiting program.")
			return
		default:
			color.Yellow("Invalid option. Please try again.")
		}
	}
}
