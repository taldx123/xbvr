package api

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"

	"github.com/xbapps/xbvr/pkg/common"
)

// PornolabResult represents a single torrent search result from PornoLab via Jackett.
type PornolabResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Size        int64  `json:"size"`
	SizeStr     string `json:"size_str"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	TopicURL    string `json:"topic_url"`
	DownloadURL string `json:"download_url"`
	Magnet      string `json:"magnet"`
	InfoHash    string `json:"info_hash"`
	Date        string `json:"date"`
}

// PornolabSearchResponse represents the API response for a PornoLab search.
type PornolabSearchResponse struct {
	Status  string           `json:"status"`
	Source  string           `json:"source"`
	Results []PornolabResult `json:"results"`
}

// PornolabMagnetResponse represents the API response containing a generated magnet link.
type PornolabMagnetResponse struct {
	Magnet   string `json:"magnet"`
	InfoHash string `json:"info_hash"`
}

// PornolabResource provides HTTP endpoints for searching and downloading from PornoLab via Jackett.
type PornolabResource struct{}

type cachedMagnet struct {
	Magnet   string `json:"magnet"`
	InfoHash string `json:"info_hash"`
}

var (
	topicIDRegex             = regexp.MustCompile(`[?&]t=(\d+)`)
	pornolabMagnetMu         sync.RWMutex
	pornolabMagnetCache      = make(map[string]cachedMagnet)
	pornolabMagnetDiskLoaded bool
	pornolabMagnetSem        = make(chan struct{}, 4)
)

func getMagnetCacheFilePath() string {
	if common.AppDir == "" {
		return ""
	}
	return filepath.Join(common.AppDir, "pornolab_magnets.json")
}

func ensureMagnetCacheLoaded() {
	pornolabMagnetMu.Lock()
	defer pornolabMagnetMu.Unlock()
	if pornolabMagnetDiskLoaded {
		return
	}
	pornolabMagnetDiskLoaded = true
	p := getMagnetCacheFilePath()
	if p == "" {
		return
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var diskMap map[string]cachedMagnet
	if err := json.Unmarshal(raw, &diskMap); err == nil {
		for k, v := range diskMap {
			pornolabMagnetCache[k] = v
		}
	}
}

func saveMagnetCacheToDisk() {
	p := getMagnetCacheFilePath()
	if p == "" {
		return
	}
	pornolabMagnetMu.RLock()
	raw, err := json.MarshalIndent(pornolabMagnetCache, "", "  ")
	pornolabMagnetMu.RUnlock()
	if err != nil {
		return
	}
	_ = os.WriteFile(p, raw, 0644)
}

// WebService returns the restful WebService for PornoLab integration.
func (i PornolabResource) WebService() *restful.WebService {
	tags := []string{"Pornolab"}

	ws := new(restful.WebService)
	ws.Path("/api/pornolab").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/search").To(i.searchPornolab).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(PornolabSearchResponse{}))

	ws.Route(ws.GET("/download").To(i.downloadTorrent).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/magnet").To(i.getMagnetLink).
		Metadata(restfulspec.KeyOpenAPITags, tags).
		Writes(PornolabMagnetResponse{}))

	return ws
}

func (i PornolabResource) searchPornolab(req *restful.Request, resp *restful.Response) {
	query := strings.TrimSpace(req.QueryParameter("q"))
	if query == "" {
		resp.WriteHeaderAndEntity(http.StatusOK, PornolabSearchResponse{
			Status:  "",
			Source:  "jackett",
			Results: []PornolabResult{},
		})
		return
	}

	jackettKey := strings.TrimSpace(os.Getenv("JACKETT_API_KEY"))
	jackettURL := strings.TrimSpace(os.Getenv("JACKETT_URL"))
	if jackettURL == "" {
		jackettURL = "http://jackett:9117"
	}

	if jackettKey == "" {
		resp.WriteHeaderAndEntity(http.StatusOK, PornolabSearchResponse{
			Status:  "Jackett is not configured. Set JACKETT_API_KEY in docker/.env.",
			Source:  "jackett",
			Results: []PornolabResult{},
		})
		return
	}

	results, err := searchViaJackett(jackettURL, jackettKey, query)
	if err != nil {
		resp.WriteHeaderAndEntity(http.StatusOK, PornolabSearchResponse{
			Status:  fmt.Sprintf("Jackett search failed: %v", err),
			Source:  "jackett",
			Results: []PornolabResult{},
		})
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, PornolabSearchResponse{
		Status:  "",
		Source:  "jackett",
		Results: results,
	})
}

type jackettResponse struct {
	Results []struct {
		Title        string `json:"Title"`
		CategoryDesc string `json:"CategoryDesc"`
		Size         int64  `json:"Size"`
		Seeders      int    `json:"Seeders"`
		Peers        int    `json:"Peers"`
		Details      string `json:"Details"`
		Guid         string `json:"Guid"`
		Link         string `json:"Link"`
		MagnetUri    string `json:"MagnetUri"`
		InfoHash     string `json:"InfoHash"`
		PublishDate  string `json:"PublishDate"`
	} `json:"Results"`
}

func searchViaJackett(jackettURL, apiKey, query string) ([]PornolabResult, error) {
	ensureMagnetCacheLoaded()

	endpoint := fmt.Sprintf("%s/api/v2.0/indexers/pornolab/results?apikey=%s&Query=%s",
		strings.TrimRight(jackettURL, "/"),
		url.QueryEscape(apiKey),
		url.QueryEscape(query),
	)

	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	httpResp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 512))
		return nil, fmt.Errorf("jackett HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	var jResp jackettResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&jResp); err != nil {
		return nil, err
	}

	results := make([]PornolabResult, 0, len(jResp.Results))
	for _, item := range jResp.Results {
		topicURL := item.Details
		if topicURL == "" {
			topicURL = item.Guid
		}
		topicID := ""
		if m := topicIDRegex.FindStringSubmatch(topicURL); len(m) > 1 {
			topicID = m[1]
		} else if m := topicIDRegex.FindStringSubmatch(item.Guid); len(m) > 1 {
			topicID = m[1]
		}
		leechers := item.Peers - item.Seeders
		if leechers < 0 {
			leechers = item.Peers
		}
		dateStr := ""
		if len(item.PublishDate) >= 10 {
			dateStr = item.PublishDate[:10]
		}

		res := PornolabResult{
			ID:          topicID,
			Title:       item.Title,
			Category:    item.CategoryDesc,
			Size:        item.Size,
			Seeders:     item.Seeders,
			Leechers:    leechers,
			TopicURL:    topicURL,
			DownloadURL: item.Link,
			Magnet:      item.MagnetUri,
			InfoHash:    item.InfoHash,
			Date:        dateStr,
		}

		if res.Magnet == "" && topicID != "" {
			pornolabMagnetMu.RLock()
			if cached, ok := pornolabMagnetCache[topicID]; ok {
				res.Magnet = cached.Magnet
				res.InfoHash = cached.InfoHash
			}
			pornolabMagnetMu.RUnlock()
		}

		results = append(results, res)
	}

	sort.Slice(results, func(a, b int) bool {
		if results[a].Date != results[b].Date {
			return results[a].Date > results[b].Date
		}
		return results[a].Seeders > results[b].Seeders
	})

	return results, nil
}

func fetchTorrentBytes(dlURL string) ([]byte, error) {
	if dlURL == "" {
		return nil, fmt.Errorf("missing download url")
	}

	client := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	httpResp, err := client.Get(dlURL)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 256))
		return nil, fmt.Errorf("Jackett torrent download failed (HTTP %d, PornoLab daily limit of 5/day may be reached): %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(httpResp.Body)
}

func resolveMagnetLink(dlURL, topicID, title string) (string, string, error) {
	ensureMagnetCacheLoaded()

	if topicID != "" {
		pornolabMagnetMu.RLock()
		if cached, ok := pornolabMagnetCache[topicID]; ok && cached.Magnet != "" {
			pornolabMagnetMu.RUnlock()
			return cached.Magnet, cached.InfoHash, nil
		}
		pornolabMagnetMu.RUnlock()
	}

	pornolabMagnetSem <- struct{}{}
	defer func() { <-pornolabMagnetSem }()

	if topicID != "" {
		pornolabMagnetMu.RLock()
		if cached, ok := pornolabMagnetCache[topicID]; ok && cached.Magnet != "" {
			pornolabMagnetMu.RUnlock()
			return cached.Magnet, cached.InfoHash, nil
		}
		pornolabMagnetMu.RUnlock()
	}

	data, err := fetchTorrentBytes(dlURL)
	if err != nil {
		return "", "", err
	}

	magnet, infoHash, err := buildMagnetFromTorrent(data, title)
	if err != nil {
		return "", "", err
	}

	if topicID != "" && magnet != "" {
		pornolabMagnetMu.Lock()
		pornolabMagnetCache[topicID] = cachedMagnet{
			Magnet:   magnet,
			InfoHash: infoHash,
		}
		pornolabMagnetMu.Unlock()
		saveMagnetCacheToDisk()
	}

	return magnet, infoHash, nil
}

func (i PornolabResource) downloadTorrent(req *restful.Request, resp *restful.Response) {
	dlURL := strings.TrimSpace(req.QueryParameter("url"))
	topicID := strings.TrimSpace(req.QueryParameter("id"))

	data, err := fetchTorrentBytes(dlURL)
	if err != nil {
		resp.WriteErrorString(http.StatusBadGateway, err.Error())
		return
	}

	filename := "pornolab.torrent"
	if topicID != "" {
		filename = fmt.Sprintf("pornolab-%s.torrent", topicID)
	}

	resp.Header().Set("Content-Type", "application/x-bittorrent")
	resp.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	resp.WriteHeader(http.StatusOK)
	_, _ = io.Copy(resp.ResponseWriter, bytes.NewReader(data))
}

func (i PornolabResource) getMagnetLink(req *restful.Request, resp *restful.Response) {
	dlURL := strings.TrimSpace(req.QueryParameter("url"))
	topicID := strings.TrimSpace(req.QueryParameter("id"))
	title := strings.TrimSpace(req.QueryParameter("title"))

	magnet, infoHash, err := resolveMagnetLink(dlURL, topicID, title)
	if err != nil {
		resp.WriteErrorString(http.StatusBadGateway, err.Error())
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, PornolabMagnetResponse{
		Magnet:   magnet,
		InfoHash: infoHash,
	})
}

func bencodeReadString(data []byte, pos int) (string, int, error) {
	if pos >= len(data) || data[pos] < '0' || data[pos] > '9' {
		return "", pos, fmt.Errorf("expected bencode string at %d", pos)
	}
	colon := bytes.IndexByte(data[pos:], ':')
	if colon < 0 {
		return "", pos, fmt.Errorf("missing colon in bencode string")
	}
	length, err := strconv.Atoi(string(data[pos : pos+colon]))
	if err != nil || length < 0 {
		return "", pos, fmt.Errorf("invalid bencode string length")
	}
	start := pos + colon + 1
	end := start + length
	if end > len(data) {
		return "", pos, fmt.Errorf("bencode string exceeds data bounds")
	}
	return string(data[start:end]), end, nil
}

func bencodeSkip(data []byte, pos int) (int, error) {
	if pos >= len(data) {
		return pos, fmt.Errorf("unexpected EOF in bencode")
	}
	switch data[pos] {
	case 'i':
		end := bytes.IndexByte(data[pos:], 'e')
		if end < 0 {
			return pos, fmt.Errorf("unterminated bencode int")
		}
		return pos + end + 1, nil
	case 'l', 'd':
		pos++
		for pos < len(data) && data[pos] != 'e' {
			var err error
			pos, err = bencodeSkip(data, pos)
			if err != nil {
				return pos, err
			}
		}
		if pos >= len(data) {
			return pos, fmt.Errorf("unterminated bencode list/dict")
		}
		return pos + 1, nil
	default:
		_, nextPos, err := bencodeReadString(data, pos)
		return nextPos, err
	}
}

func bencodeCollectStrings(data []byte, pos int, out *[]string) (int, error) {
	if pos >= len(data) {
		return pos, nil
	}
	switch data[pos] {
	case 'l':
		pos++
		for pos < len(data) && data[pos] != 'e' {
			var err error
			pos, err = bencodeCollectStrings(data, pos, out)
			if err != nil {
				return pos, err
			}
		}
		if pos < len(data) {
			pos++
		}
		return pos, nil
	default:
		if data[pos] >= '0' && data[pos] <= '9' {
			s, nextPos, err := bencodeReadString(data, pos)
			if err != nil {
				return pos, err
			}
			if s != "" {
				*out = append(*out, s)
			}
			return nextPos, nil
		}
		return bencodeSkip(data, pos)
	}
}

func extractInfoName(infoBytes []byte) string {
	if len(infoBytes) < 2 || infoBytes[0] != 'd' {
		return ""
	}
	pos := 1
	for pos < len(infoBytes) && infoBytes[pos] != 'e' {
		key, valStart, err := bencodeReadString(infoBytes, pos)
		if err != nil {
			return ""
		}
		valEnd, err := bencodeSkip(infoBytes, valStart)
		if err != nil {
			return ""
		}
		if key == "name" {
			if name, _, nErr := bencodeReadString(infoBytes, valStart); nErr == nil && strings.TrimSpace(name) != "" {
				return strings.TrimSpace(name)
			}
		}
		pos = valEnd
	}
	return ""
}

func buildMagnetFromTorrent(data []byte, fallbackTitle string) (string, string, error) {
	if len(data) == 0 || data[0] != 'd' {
		return "", "", fmt.Errorf("invalid torrent file payload")
	}

	var infoBytes []byte
	var trackers []string
	seenTrackers := map[string]bool{}
	addTracker := func(tr string) {
		tr = strings.TrimSpace(tr)
		if tr != "" && !seenTrackers[tr] {
			seenTrackers[tr] = true
			trackers = append(trackers, tr)
		}
	}

	// Include standard public DHT/UDP trackers used by btdig / DMM alongside tracker announces
	addTracker("udp://tracker.openbittorrent.com:80")
	addTracker("udp://tracker.opentrackr.org:1337/announce")

	pos := 1
	for pos < len(data) && data[pos] != 'e' {
		key, valStart, err := bencodeReadString(data, pos)
		if err != nil {
			return "", "", err
		}
		valEnd, err := bencodeSkip(data, valStart)
		if err != nil {
			return "", "", err
		}

		switch key {
		case "info":
			infoBytes = data[valStart:valEnd]
		case "announce":
			if tr, _, err := bencodeReadString(data, valStart); err == nil {
				addTracker(tr)
			}
		case "announce-list":
			var list []string
			_, _ = bencodeCollectStrings(data, valStart, &list)
			for _, tr := range list {
				addTracker(tr)
			}
		}
		pos = valEnd
	}

	if len(infoBytes) == 0 {
		return "", "", fmt.Errorf("missing info dictionary in torrent")
	}

	sum := sha1.Sum(infoBytes)
	infoHash := hex.EncodeToString(sum[:])

	dn := extractInfoName(infoBytes)
	if dn == "" {
		dn = fallbackTitle
	}

	var sb strings.Builder
	sb.WriteString("magnet:?xt=urn:btih:")
	sb.WriteString(infoHash)
	if dn != "" {
		sb.WriteString("&dn=")
		sb.WriteString(url.QueryEscape(dn))
	}
	for _, tr := range trackers {
		sb.WriteString("&tr=")
		sb.WriteString(url.QueryEscape(tr))
	}

	return sb.String(), infoHash, nil
}
