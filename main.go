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
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	gomail "gopkg.in/mail.v2"
)

// LAT,LNG for which to check the forecast
var LAT_LNG_MAP = map[string][2]float64{
	"Plaz":       {38.280795, 21.746126},
	"South Park": {38.234688, 21.724288},
}

// Best workout hours
var workoutHours = [5]int{16, 17, 18, 19, 20}

var skyGlyphs = map[string]string{
	"Sunny":                         "☀️",
	"Clear":                         "🌙",
	"Cloudy":                        "☁️",
	"Partly Cloudy":                 "⛅",
	"Patchy rain nearby":            "🌧️",
	"Light rain shower":             "🌧️",
	"Light rain":                    "🌧️",
	"Moderate rain":                 "🌧️",
	"Light Drizzle":                 "🌧️",
	"Moderate or heavy rain shower": "🌧️",
	"Overcast":                      "☁️",
	// Add more as needed…
}

// constructURL builds the full API request URL for the weather forecast.
// Parameters:
//
//	baseURL: The base endpoint of the weather API.
//	apiKey:  The API key for authentication.
//	lat:     Latitude of the location.
//	lng:     Longitude of the location.
//	days:    Number of days to forecast.
//
// Returns:
//
//	The complete URL string with query parameters for the API request.
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

// getForecastResult sends an HTTP GET request to the provided URL and returns the response.
// Parameters:
//
//	fullURL: The complete URL string for the weather API request.
//
// Returns:
//
//	*http.Response: The HTTP response from the API if successful.
//	error: An error if the request fails or the response status is not OK.
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

// getForecastForRemainingDaysOfWeek extracts the forecast for each day until the end of the week from the API result.
//
// Parameters:
//
//	weatherApiResult: The decoded JSON response from the weather API as a map.
//
// Returns:
//
//	A map where each key is a day (with date) and the value is a map of workout hours to weather summaries.
func getForecastForRemainingDaysOfWeek(weatherApiResult map[string]any) map[string]any {
	// I want to return the forecast for each day until the end of the week
	var forecasts = map[string]any{}

	// 1. Get the current day
	currentDay := time.Now().Weekday()
	lastForecastDay := int(currentDay) + 3
	if int(currentDay) == 6 || int(currentDay) == 7 {
		lastForecastDay = 7
	}

	fmt.Println(int(currentDay))
	// 2. Loop through the next 3 days
	for i := int(currentDay); i <= lastForecastDay; i++ {
		dayForecast := weatherApiResult["forecast"].(map[string]any)["forecastday"].([]any)[i-int(currentDay)].(map[string]any)

		dayForecastDate := dayForecast["date"].(string)

		hoursForecast := dayForecast["hour"].([]any)
		var hoursForecastMap = map[string]any{}
		// 3. For each day, check the hourly forecast
		for workoutHour := workoutHours[0]; workoutHour <= workoutHours[len(workoutHours)-1]; workoutHour++ {
			builder := &strings.Builder{}
			hourForecast := hoursForecast[workoutHour].(map[string]any)["condition"].(map[string]any)["text"].(string)
			fmt.Fprintf(builder, "%s", hourForecast)

			hourTemp := strconv.FormatFloat(hoursForecast[workoutHour].(map[string]any)["temp_c"].(float64), 'f', 0, 64)
			fmt.Fprintf(builder, " %s °C", hourTemp)

			hoursForecastMap[fmt.Sprintf("%d:00", workoutHour)] = builder.String()
		}

		forecastMapKey := fmt.Sprintf("%s (%s)", time.Weekday(i%7).String(), dayForecastDate)

		forecasts[forecastMapKey] = hoursForecastMap
	}

	return forecasts
}

// createEmailText formats the forecast data into a human-readable email body.
//
// Parameters:
//
//	forecasts: A map where each key is a day (with date) and the value is a map of workout hours to weather summaries.
//
// Returns:
//
//	A string containing the formatted email text with weather information for each day and hour.
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
			hourForecast := hoursMap[hour].(string)
			hourWeather := hourForecast[:len(hourForecast)-6]
			hourTemp := hourForecast[len(hourWeather):]

			trimmed := strings.TrimSpace(hourWeather)
			icon := skyGlyphs[trimmed]
			// fmt.Fprintf(builder, "  %s: %s\n", hour, icon)
			fmt.Fprintf(builder, "  %s: %s %s (%s)\n", hour, trimmed, icon, hourTemp)
		}
	}

	emailText := builder.String()
	return emailText
}

// sendEmail sends an email with the provided text to the specified recipient.
//
// Parameters:
//
//	from: The sender's email address.
//	to: The recipient's email address.
//	password: The sender's email password for SMTP authentication.
//	text: The body text of the email.
//
// Returns:
//
//	error: An error if the email fails to send, nil if successful.
func sendEmail(from, to, password, text string) error {
	message := gomail.NewMessage()

	message.SetHeader("From", from)
	message.SetHeader("To", to)
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

// sortSliceByDateInParentheses sorts a slice of day strings by their embedded dates in parentheses.
//
// Parameters:
//
//	days: A slice of strings where dates are enclosed in parentheses at the end of each string.
//
// Returns:
//
//	A new sorted slice of day strings in chronological order based on the dates.
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

// calculateRemainingDaysInWeek calculates how many days are left in the current week.
//
// Returns:
//
//	The number of remaining days in the week (including today).
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
	to := os.Getenv("to")
	password := os.Getenv("password")

	err = sendEmail(from, to, password, emailText)
	if err != nil {
		log.Fatalf("Error sending email: %v", err)
		return
	}

}
