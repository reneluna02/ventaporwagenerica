package maps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const (
	geocodeURL    = "https://maps.googleapis.com/maps/api/geocode/json"
	staticMapURL  = "https://maps.googleapis.com/maps/api/staticmap"
	streetViewURL = "https://maps.googleapis.com/maps/api/streetview"
)

// Client es un cliente para interactuar con las APIs de Google Maps.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient crea un nuevo cliente de Google Maps.
func NewClient() (*Client, error) {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("la variable de entorno GOOGLE_MAPS_API_KEY no está configurada")
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

// GeocodeResponse es la estructura de la respuesta de la API de Geocoding.
type GeocodeResponse struct {
	Results []struct {
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}

// Geocode convierte una dirección en coordenadas (latitud y longitud).
func (c *Client) Geocode(address string) (lat, lng float64, err error) {
	req, err := http.NewRequest("GET", geocodeURL, nil)
	if err != nil {
		return 0, 0, err
	}

	q := req.URL.Query()
	q.Add("address", address)
	q.Add("key", c.apiKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var geoResp GeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return 0, 0, err
	}

	if len(geoResp.Results) == 0 {
		return 0, 0, fmt.Errorf("no se encontraron resultados para la dirección: %s", address)
	}

	loc := geoResp.Results[0].Geometry.Location
	return loc.Lat, loc.Lng, nil
}

// GenerateStaticMapURL genera una URL para un mapa estático.
func (c *Client) GenerateStaticMapURL(lat, lng float64) string {
	params := url.Values{}
	params.Set("center", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("zoom", "17")
	params.Set("size", "600x300")
	params.Set("maptype", "roadmap")
	params.Set("markers", fmt.Sprintf("color:red|label:S|%f,%f", lat, lng))
	params.Set("key", c.apiKey)
	return staticMapURL + "?" + params.Encode()
}

// GenerateStreetViewURL genera una URL para una imagen de Street View.
func (c *Client) GenerateStreetViewURL(lat, lng float64) string {
	params := url.Values{}
	params.Set("size", "600x300")
	params.Set("location", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("heading", "151.78")
	params.Set("pitch", "-0.76")
	params.Set("key", c.apiKey)
	return streetViewURL + "?" + params.Encode()
}
