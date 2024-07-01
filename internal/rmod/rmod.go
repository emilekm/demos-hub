package rmod

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type licenseResponse struct {
	Valid bool   `json:"Valid"`
	User  string `json:"User"`
}

type licenseError struct {
	ErrorCode int    `json:"ErrorCode"`
	ErrorMsg  string `json:"ErrorMsg"`
}

func (e licenseError) Error() string {
	return fmt.Sprintf("License error: %d - %s", e.ErrorCode, e.ErrorMsg)
}

func ValidateLicense(ip, port, license string) (bool, error) {
	u, err := url.Parse("http://www.realitymod.com/forum/lcp_validate.php")
	if err != nil {
		return false, err
	}

	q := u.Query()
	q.Set("action", "server")
	q.Set("game", "prbf2")
	q.Set("key", license)
	q.Set("ip", ip)
	q.Set("port", port)

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		var lErr licenseError
		err = json.Unmarshal(content, &lErr)
		if err != nil {
			return false, err
		}

		return false, licenseError{lErr.ErrorCode, lErr.ErrorMsg}
	}

	var lResp licenseResponse
	err = json.Unmarshal(content, &lResp)
	if err != nil {
		return false, err
	}

	return lResp.Valid, nil
}
