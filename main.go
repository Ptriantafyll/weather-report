package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	gomail "gopkg.in/mail.v2"
)

var LAT_LNG_MAP = map[string][2]float64{
	"Plaz":       {38.280795, 21.746126},
	"South Park": {38.234688, 21.724288},
}

var skyGlyphs = map[string]string{
	"Sunny":              "☀️",
	"Clear":              "🌙",
	"Cloudy":             "☁️",
	"Partly Cloudy":      "⛅",
	"Patchy rain nearby": "🌧️",
	"Light rain shower":  "🌧️",
	"Moderate rain":      "🌧️",
	"Overcast":           "☁️",
	// Add more as needed…
}

func constructURL(baseURL, apiKey string, lat, lng float64, days int) string {
	params := url.Values{}
	latLngString := fmt.Sprintf("%f,%f", lat, lng)

	params.Add("key", apiKey)
	params.Add("q", latLngString)
	params.Add("days", fmt.Sprint(days))
	params.Add("aqi", "no")

	fullURL := baseURL + "?" + params.Encode()
	return fullURL
}

func getForecastResult(fullURL string) (*http.Response, error) {
	response, err := http.Get(fullURL)
	if err != nil {
		log.Fatalf("Error making GET request: %v", err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		log.Fatalf("Non-OK HTTP status: %s", response.Status)
		return nil, errors.New("non-OK HTTP status")
	}

	return response, nil
}

func getForecastForRemainingDaysOfWeek(weatherApiResult map[string]any) map[string]any {
	// I want to return the forecast for each day until the end of the week
	var forecasts = map[string]any{}

	// 1. Get the current day
	currentDay := time.Now().Weekday()
	workoutHours := [5]int{16, 17, 18, 19, 20}
	// 2. Loop through the forecast days until the end of the week
	for i := int(currentDay); i < 8; i++ {
		dayForecast := weatherApiResult["forecast"].(map[string]any)["forecastday"].([]any)[i-int(currentDay)].(map[string]any)

		dayForecastDate := dayForecast["date"].(string)

		hoursForecast := dayForecast["hour"].([]any)
		var hoursForecastMap = map[string]any{}
		// 3. For each day, check the hourly forecast
		for j := workoutHours[0]; j <= workoutHours[len(workoutHours)-1]; j++ {
			hourForecast := hoursForecast[j].(map[string]any)["condition"].(map[string]any)["text"].(string)

			hoursForecastMap[fmt.Sprintf("%d:00", j)] = hourForecast
		}

		forecastMapKey := fmt.Sprintf("%s (%s)", time.Weekday(i%7).String(), dayForecastDate)

		forecasts[forecastMapKey] = hoursForecastMap
	}

	return forecasts
}

func createEmailText(forecasts map[string]any) string {
	builder := &strings.Builder{}
	days := slices.Collect(maps.Keys(forecasts))
	days = sortSliceByDateInParentheses(days)

	for _, day := range days {
		fmt.Fprintf(builder, "%s\n", day)

		hoursMap := forecasts[day].(map[string]any)

		hours := slices.Collect(maps.Keys(hoursMap))
		sort.Strings(hours)

		for _, hour := range hours {
			trimmed := strings.TrimSpace(hoursMap[hour].(string))
			icon := skyGlyphs[trimmed]
			// fmt.Fprintf(builder, "  %s: %s\n", hour, icon)
			fmt.Fprintf(builder, "  %s: %s %s\n", hour, trimmed, icon)
		}
	}

	emailText := builder.String()
	return emailText
}

func sendEmail(from, password, text string) error {
	message := gomail.NewMessage()

	message.SetHeader("From", from)
	message.SetHeader("To", from)
	message.SetHeader("Subject", "Weather Report")
	message.SetBody("text/plain", text)

	dialer := gomail.NewDialer("smtp.gmail.com", 587, from, password)

	if err := dialer.DialAndSend(message); err != nil {
		log.Fatalf("Could not send email: %v", err)
		return err
	}

	fmt.Println("Email sent successfully")
	return nil
}

func sortSliceByDateInParentheses(days []string) []string {
	sort.Slice(days, func(i, j int) bool {

		extract := func(s string) time.Time {
			start := strings.LastIndex(s, "(") + 1
			end := strings.LastIndex(s, ")")
			t, err := time.Parse("2006-01-02", s[start:end])

			if err != nil {
				log.Fatalf("Error parsing date: %v", err)
				return time.Time{}
			}
			return t
		}

		return extract(days[i]).Before(extract(days[j]))
	})

	return days
}

func calculateRemainingDaysInWeek() int {
	currentDay := time.Now().Weekday()
	return 8 - int(currentDay)%7
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("WEATHERAPI_KEY")
	if apiKey == "" {
		log.Fatalf("WEATHERAPI_KEY env variable not set")
		return
	}

	// construct full URL
	baseURL := "http://api.weatherapi.com/v1/forecast.json"
	daysToForecast := calculateRemainingDaysInWeek()
	fullURL := constructURL(baseURL, apiKey, LAT_LNG_MAP["Plaz"][0], LAT_LNG_MAP["Plaz"][1], daysToForecast)

	// get forecast
	response, err := getForecastResult(fullURL)

	if err != nil {
		log.Fatalf("Error getting forecast: %v", err)
		return
	}
	defer response.Body.Close()

	// get needed data from response
	var result map[string]any
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		log.Fatalf("Error decoding JSON response: %v", err)
		return
	}

	forecasts := getForecastForRemainingDaysOfWeek(result)
	emailText := createEmailText(forecasts)

	fmt.Println(emailText)

	// send email about the report
	from := os.Getenv("from")
	password := os.Getenv("password")

	err = sendEmail(from, password, emailText)
	if err != nil {
		log.Fatalf("Error sending email: %v", err)
		return
	}

}
