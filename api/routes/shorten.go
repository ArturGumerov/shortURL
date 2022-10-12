package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/arturgumerov/shortURL/database"
	"github.com/arturgumerov/shortURL/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL            string        `json:"url"`
	CustomShort    string        `json:"short"`
	Expiry         time.Duration `json:"expiry"`
	XRateRemining  int           `json:"rate_remining"`
	XRateLimitRest int           `json:"rate_limit_rest"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}
	return nil

	//implement rate limiting

	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60 *time.Second).Err()
	}else{
		val,_=r2.Get(database.Ctx,c.IP().Result())
		valInt,_:= strconv.Atoi(val)
		if valInt <= 0{
			limit,_:=r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":"rate limit exceeded",
				"rate_limit_rest": limit / time.Nanosecond
			})
		}
	}

	//check the input if an actual URL

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "URL not valid"})
	}
	//check for domain error

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Domain error"})
	}

	//enforce https

	body.URL = helpers.EnforceHTTPS(body.URL)


	r2.Decr(database.Ctx, c.IP())
}
