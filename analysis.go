// Package analysis contains functions for analysis of stock data
package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math"
)

type Stock struct {
	Symbol           string `json:"t"`
	Exchange         string `json:"e"`
	Name             string `json:"name"`
	Change           string `json:"c"`
	Close            string `json:"l"`
	PercentageChange string `json:"cp"`
	CloseChange      string `json:"c"`
	Open             string `json:"op"`
	High             string `json:"hi"`
	Low              string `json:"lo"`
	Volume           string `json:"vo"`
	AverageVolume    string `json:"avvo"`
	High52           string `json:"hi52"`
	Low52            string `json:"lo52"`
	MarketCap        string `json:"mc"`
	EPS              string `json:"eps"`
	Shares           string `json:"shares"`
	Beta             string `json:"beta"`
}

type TrendingStock struct {
	*Stock
	TrendingDirection string
	TrendingStrength  int
	Volatility        float64
	VolatilityPerc    float64
}

//@TODO We can use sorting to show the top movers etc
// http://nerdyworm.com/blog/2013/05/15/sorting-a-slice-of-structs-in-go/

func CalculateTrends(configuration Configuration, stockList []Stock, db *sql.DB, grouping string, trendDepth int) (trendingStocks []TrendingStock) {
	db, err := sql.Open("sqlite3", configuration.SQLite3File)
	if err != nil {
		fmt.Println("Could not connect to database")
		return
	}

	fmt.Println("\t\t\tChecking for trends")
	trendingStocks = make([]TrendingStock, 0)
	for i := range stockList {
		stock := stockList[i]

		// Prepare statement for inserting data
		// Empty vars for initialisation
		rows, err := db.Query("")
		// Need to manually put string in as it doesnt parse grouping correctly
		switch grouping {
		// SQL must get only the close price
		case "day":
			rows, err = db.Query("SELECT `close`, `avgVolume` FROM `st_data` WHERE `symbol` = ? AND `hour` = 17 AND `minute` = 15 ORDER BY `id` DESC LIMIT ?", stock.Symbol, trendDepth)
			break
		case "hour":
			rows, err = db.Query("SELECT `close`, `avgVolume` FROM `st_data` WHERE `symbol` = ? AND `hour` = 17 AND `minute` = 15 ORDER BY `id` DESC LIMIT ?", stock.Symbol, trendDepth)
			break
		}
		//rows, err := db.Query("SELECT `close`, `volume` FROM `st_data` WHERE `symbol` = ? LIMIT 3", stock.Symbol)
		if err != nil {
			fmt.Println("Error with select query: " + err.Error())
		}
		defer rows.Close()

		allCloses := make([]float64, 0)
		allVolumes := make([]float64, 0)
		count := 0
		for rows.Next() {
			var stockClose float64
			var stockVolume float64
			if err := rows.Scan(&stockClose, &stockVolume); err != nil {
				log.Fatal(err)
			}
			allCloses = append(allCloses, stockClose)
			allVolumes = append(allVolumes, stockVolume)
			count++
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		if count >= 3 {
			if doTrendCalculation(allCloses, allVolumes, "up", stock.Symbol, grouping, trendDepth) {
				fmt.Printf("\t\t\tTrend UP for %s\n", stock.Symbol)
				volatility, volatilityPerc := calculateStdDev(configuration, db, stock.Symbol, 2)

				trendingStock := TrendingStock{&stock, "up", 0, volatility, volatilityPerc}
				trendingStocks = append(trendingStocks, trendingStock)
			} else if doTrendCalculation(allCloses, allVolumes, "down", stock.Symbol, grouping, trendDepth) {
				fmt.Printf("\t\t\tTrend DOWN for %s\n", stock.Symbol)
				volatility, volatilityPerc := calculateStdDev(configuration, db, stock.Symbol, 2)

				trendingStock := TrendingStock{&stock, "down", 0, volatility, volatilityPerc}
				trendingStocks = append(trendingStocks, trendingStock)
			}
		}

	}
	defer db.Close()

	return
}

func doTrendCalculation(closes []float64, volumes []float64, trendType string, symbol string, grouping string, trendDepth int) (trending bool) {
	/*@TODO
	  Currently a simple analysis is done on daily stock data. This analysis is to identify trending stocks, with a trend being identified by:
	  - A price increase (or decrease) each day for three days
	  - A volume increase (or decrease) over two of the three days
	*/
	fmt.Printf("\t\t\t\tChecking %s trends with data: price: %f, %f, %f and volume: %f, %f, %f\n", symbol, closes[0], closes[1], closes[2], volumes[0], volumes[1], volumes[2])
	switch trendType {
	case "up":
		if grouping == "day" {
			if closes[0] > closes[1] && closes[1] > closes[2] && (volumes[0] > volumes[2] || volumes[0] > volumes[1]) {
				return true
			}
		} else if grouping == "hour" {
			// For now we aim for 3 periods
			if closes[0] > closes[1] && closes[1] > closes[2] && (volumes[0] > volumes[2] || volumes[0] > volumes[1]) {
				return true
			}
		}
		break
	case "down":
		if grouping == "day" {
			if closes[0] < closes[1] && closes[1] < closes[2] && (volumes[0] > volumes[2] || volumes[0] > volumes[1]) {
				return true
			}
		} else if grouping == "hour" {
			// For now we aim for 3 periods
			if closes[0] < closes[1] && closes[1] < closes[2] && (volumes[0] > volumes[2] || volumes[0] > volumes[1]) {
				return true
			}
		}
		break
	}

	return false
}

func calculateStdDev(configuration Configuration, db *sql.DB, symbol string, decimalPlaces int) (volatility float64, volatilityPerc float64) {
	fmt.Println("Calculating standard deviation for symbol " + symbol)

	db, err := sql.Open("sqlite3", configuration.SQLite3File)
	if err != nil {
		fmt.Println("Could not connect to database")
		return
	}

	// Get all closes for given stock
	rows, err := db.Query("SELECT `close` FROM `st_data` WHERE `symbol` = ? GROUP BY `day` LIMIT 365", symbol) // Default is one year's data
	if err != nil {
		fmt.Println("Error with select query: " + err.Error())
	}
	defer rows.Close()

	allCloses := make([]float64, 0)
	var totalCloses float64
	count := 0.
	for rows.Next() {
		var stockClose float64
		if err := rows.Scan(&stockClose); err != nil {
			log.Fatal(err)
		}
		totalCloses += stockClose

		fmt.Printf("Close at count %f is %f\n", count, stockClose)
		allCloses = append(allCloses, stockClose)
		count++
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total closes %f\n", count)

	// Calculate mean
	mean := totalCloses / count
	fmt.Printf("Mean is %f\n", mean)

	// Get all deviations
	deviationsSquare := 0.
	for _, cl := range allCloses {
		dev := cl - mean
		deviationsSquare += dev * dev
	}
	fmt.Printf("Deviations square is %f\n", deviationsSquare)

	// Calculate average square of deviations
	devSquareAvg := deviationsSquare / count
	fmt.Printf("Deviations square average is %f\n", devSquareAvg)

	// Volatility is sqrt
	volatility = math.Sqrt(devSquareAvg)

	fmt.Printf("Volatility of stock %s is %f\n", symbol, volatility)

	// Make volatility a % so we can judge
	volatilityPerc = (volatility / allCloses[int(count)-1]) * 100
	fmt.Printf("Volatility of stock %s as percentage is %f\n", symbol, volatilityPerc)

	// Round the volatility
	if decimalPlaces != 0 {
		volatility = RoundDown(volatility, decimalPlaces)
		volatilityPerc = RoundDown(volatilityPerc, decimalPlaces)
	}

	defer db.Close()

	return
}

func RoundDown(input float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * input
	round = math.Floor(digit)
	newVal = round / pow
	return
}
