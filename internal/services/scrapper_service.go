package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// contiene los metadatos extraidos de la url
type MetadataResult struct {
	Title       string
	Description string
	Favicon     string
	Error       error
}

// maneja la extracion de metadatos de la url
type ScraperService struct {
	client  *http.Client
	timeout time.Duration
}

// instancia del servicio de scrapper

func NewScraperService() *ScraperService {
	return &ScraperService{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("Demasiados redirects")
				}

				return nil
			},
		},
		timeout: 10 * time.Second,
	}
}

// ScrapeMetadata extrae título, descripción y favicon de una URL
func (s *ScraperService) ScrapeMetadata(targetURL string) *MetadataResult {
	result := &MetadataResult{}

	// Validar la URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		result.Error = fmt.Errorf("URL inválida: %w", err)
		return result
	}

	// Asegurar que la URL tenga un esquema
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
		targetURL = parsedURL.String()
	}

	// Hacer la petición HTTP
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("error creando petición: %w", err)
		return result
	}

	// Agregar headers para parecer un navegador real
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	// Ejecutar la petición
	resp, err := s.client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("error haciendo petición: %w", err)
		return result
	}
	defer resp.Body.Close()

	// Verificar el status code
	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Errorf("status code inválido: %d", resp.StatusCode)
		return result
	}

	// Parsear el HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("error parseando HTML: %w", err)
		return result
	}

	// Extraer metadatos
	result.Title = s.extractTitle(doc)
	result.Description = s.extractDescription(doc)
	result.Favicon = s.extractFavicon(doc, parsedURL)

	return result
}

// extractTitle extrae el título de la página
// Prioridad: og:title > twitter:title > <title>
func (s *ScraperService) extractTitle(doc *goquery.Document) string {
	// Intentar Open Graph title
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists && title != "" {
		return strings.TrimSpace(title)
	}

	// Intentar Twitter title
	if title, exists := doc.Find("meta[name='twitter:title']").Attr("content"); exists && title != "" {
		return strings.TrimSpace(title)
	}

	// Intentar tag <title>
	if title := doc.Find("title").First().Text(); title != "" {
		return strings.TrimSpace(title)
	}

	return ""
}

// extractDescription extrae la descripción de la página
// Prioridad: og:description > twitter:description > meta description
func (s *ScraperService) extractDescription(doc *goquery.Document) string {
	// Intentar Open Graph description
	if desc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	// Intentar Twitter description
	if desc, exists := doc.Find("meta[name='twitter:description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	// Intentar meta description estándar
	if desc, exists := doc.Find("meta[name='description']").Attr("content"); exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	return ""
}

// extractFavicon extrae la URL del favicon
// Busca en múltiples ubicaciones posibles
func (s *ScraperService) extractFavicon(doc *goquery.Document, baseURL *url.URL) string {
	var faviconPath string

	// Lista de selectores de favicon en orden de preferencia
	selectors := []struct {
		selector string
		attr     string
	}{
		// Favicon de alta resolución
		{"link[rel='apple-touch-icon']", "href"},
		{"link[rel='icon'][sizes='192x192']", "href"},
		{"link[rel='icon'][sizes='180x180']", "href"},
		// Favicon estándar
		{"link[rel='icon']", "href"},
		{"link[rel='shortcut icon']", "href"},
		// Open Graph image como fallback
		{"meta[property='og:image']", "content"},
	}

	// Buscar favicon usando los selectores
	for _, sel := range selectors {
		if path, exists := doc.Find(sel.selector).First().Attr(sel.attr); exists && path != "" {
			faviconPath = path
			break
		}
	}

	// Si no se encontró favicon, usar el favicon por defecto
	if faviconPath == "" {
		faviconPath = "/favicon.ico"
	}

	// Convertir a URL absoluta
	faviconURL, err := url.Parse(faviconPath)
	if err != nil {
		return ""
	}

	// Si es una URL relativa, resolverla
	if !faviconURL.IsAbs() {
		faviconURL = baseURL.ResolveReference(faviconURL)
	}

	return faviconURL.String()
}

// ScrapeMetadataAsync ejecuta el scraping de forma asíncrona con timeout
func (s *ScraperService) ScrapeMetadataAsync(targetURL string, timeout time.Duration) *MetadataResult {
	resultChan := make(chan *MetadataResult, 1)

	go func() {
		resultChan <- s.ScrapeMetadata(targetURL)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-time.After(timeout):
		return &MetadataResult{
			Error: errors.New("timeout: el scraping tardó demasiado tiempo"),
		}
	}
}
