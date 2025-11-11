# Weather report

This project is a weather report designed to show the best possible days for an outside workout throughout the week.

## Prerequisites

1. `go get -u github.com/joho/godotenv`
2. [weatherapi.com](https://www.weatherapi.com/) api key (free for 3-day forecast)

## How it works

The project calls [weatherapi.com](https://www.weatherapi.com/) by giving latitude and longitude and sends an email with the weather condition and temperature (°C) for the next 3 days (including current day) for hours 16:00-20:00 (the best hours for an outside workout)

The hours, as well as the (latitute, longitude) are stored in the global variables LAT_LNG_MAP and workoutHours.

Variables stored on .env:

* WEATHERAPI_KEY - your weatherapi.com key
* from           - the email from which you are sending
* to             - the email to which you are sending
* password       - email app password for the `from` email (e.g. [generating gmail app password](https://support.google.com/mail/thread/205453566/how-to-generate-an-app-password?hl=en))

Example email text: 
```
Tuesday (2025-11-11)
  16:00: Sunny ☀️ (18 °C)
  17:00: Sunny ☀️ (17 °C)
  18:00: Clear 🌙 (15 °C)
  19:00: Clear 🌙 (15 °C)
  20:00: Cloudy ☁️ (14 °C)
Wednesday (2025-11-12)
  16:00: Sunny ☀️ (18 °C)
  17:00: Sunny ☀️ (16 °C)
  18:00: Clear 🌙 (15 °C)
  19:00: Clear 🌙 (14 °C)
  20:00: Clear 🌙 (14 °C)
Thursday (2025-11-13)
  16:00: Sunny ☀️ (17 °C)
  17:00: Sunny ☀️ (15 °C)
  18:00: Clear 🌙 (14 °C)
  19:00: Clear 🌙 (14 °C)
  20:00: Clear 🌙 (13 °C)
```

## Deployment

1. go build .
2. ./weather-report