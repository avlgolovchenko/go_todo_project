package rules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	taskDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", fmt.Errorf("некорректная дата: %w", err)
	}

	if repeat == "" {
		return "", errors.New("пустое правило повторения")
	}

	if strings.HasPrefix(repeat, "d ") {
		parts := strings.Split(repeat, " ")
		if len(parts) != 2 {
			return "", errors.New("некорректный формат правила d")
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 || days > 400 {
			return "", errors.New("некорректное количество дней в правиле d")
		}

		for {
			taskDate = taskDate.AddDate(0, 0, days)
			if taskDate.After(now) {
				break
			}
		}
		return taskDate.Format("20060102"), nil
	}

	if repeat == "y" {
		for {
			taskDate = taskDate.AddDate(1, 0, 0)
			if taskDate.After(now) {
				break
			}
		}
		return taskDate.Format("20060102"), nil
	}

	if strings.HasPrefix(repeat, "w ") {
		parts := strings.Split(repeat[2:], ",")
		var weekdays []time.Weekday
		for _, part := range parts {
			day, err := strconv.Atoi(part)
			if err != nil || day < 1 || day > 7 {
				return "", fmt.Errorf("некорректный день недели: %s", part)
			}
			weekdays = append(weekdays, time.Weekday(day%7))
		}

		for {
			taskDate = taskDate.AddDate(0, 0, 1)
			for _, wd := range weekdays {
				if taskDate.Weekday() == wd && taskDate.After(now) {
					return taskDate.Format("20060102"), nil
				}
			}
		}
	}

	if strings.HasPrefix(repeat, "m ") {
		parts := strings.Split(repeat[2:], " ")
		if len(parts) < 1 || len(parts) > 2 {
			return "", errors.New("некорректный формат правила m")
		}

		dayParts := strings.Split(parts[0], ",")
		var days []int
		for _, part := range dayParts {
			day, err := strconv.Atoi(part)
			if err != nil || (day < -31 || day > 31) || day == 0 {
				return "", fmt.Errorf("некорректный день месяца: %s", part)
			}
			days = append(days, day)
		}

		months := map[int]bool{}
		if len(parts) == 2 {
			monthParts := strings.Split(parts[1], ",")
			for _, part := range monthParts {
				month, err := strconv.Atoi(part)
				if err != nil || month < 1 || month > 12 {
					return "", fmt.Errorf("некорректный месяц: %s", part)
				}
				months[month] = true
			}
		}

		for {
			taskDate = taskDate.AddDate(0, 0, 1)
			day := taskDate.Day()
			month := int(taskDate.Month())

			if len(months) > 0 && !months[month] {
				continue
			}

			for _, d := range days {
				if d > 0 && day == d {
					return taskDate.Format("20060102"), nil
				} else if d < 0 {
					lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, taskDate.Location()).Day()
					if day == lastDay+d+1 {
						return taskDate.Format("20060102"), nil
					}
				}
			}
		}
	}

	return "", errors.New("неподдерживаемый формат правила")
}
