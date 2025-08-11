package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gerryyang2025/llm-news/internal/models"
	"github.com/gerryyang2025/llm-news/internal/papers"
	"github.com/gerryyang2025/llm-news/internal/scrapers"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
)

var (
	githubRepos    []models.Repository
	researchPapers []models.Paper
	lastUpdated    time.Time
	verboseLogging = false // 控制是否输出详细日志
)

func getLocalIP() string {
	// 默认IP
	defaultIP := "0.0.0.0"

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return defaultIP
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		// 排除 loopback、down 接口和虚拟接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口的 IP 地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 排除 IPv6、loopback 和 link-local 地址
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.To4() == nil {
				continue
			}

			// 找到合适的 IPv4 地址
			return ip.String()
		}
	}

	// 如果没有找到适合的IP，返回默认值
	return defaultIP
}

func main() {
	// 设置Gin为release模式，减少调试输出
	gin.SetMode(gin.ReleaseMode)

	// 自定义日志函数
	logInfo := func(format string, v ...interface{}) {
		log.Printf(format, v...)
	}

	logWarning := func(format string, v ...interface{}) {
		if verboseLogging {
			log.Printf("Warning: "+format, v...)
		}
	}

	logError := func(format string, v ...interface{}) {
		log.Printf("Error: "+format, v...)
	}

	// 记录使用自定义日志函数的信息
	logInfo("LLM News server initializing...")
	logWarning("Verbose logging is currently %t", verboseLogging)

	// Initialize the scheduler
	s := gocron.NewScheduler(time.UTC)

	// Schedule GitHub trending scraping every 1 hour
	s.Every(1).Hour().Do(func() {
		logInfo("Scraping GitHub trending repositories...")
		repos, err := scrapers.ScrapeGithubTrending()
		if err != nil {
			logError("Error scraping GitHub trending: %v", err)
			return
		}
		githubRepos = repos
		lastUpdated = time.Now()
		logInfo("Found %d trending repositories", len(repos))
	})

	// Schedule research papers scraping every 6 hours (more frequent than daily)
	s.Every(6).Hours().Do(func() {
		logInfo("Fetching latest AI research papers...")
		papers, err := papers.FetchTopPapers()
		if err != nil {
			logError("Error fetching research papers: %v", err)
			return
		}
		researchPapers = papers
		lastUpdated = time.Now()
		logInfo("Found %d research papers", len(papers))
	})

	// Start the scheduler in a separate goroutine
	s.StartAsync()

	// Run initial scraping
	logInfo("Running initial data collection...")

	// GitHub trending
	repos, err := scrapers.ScrapeGithubTrending()
	if err != nil {
		logError("Initial GitHub scraping error: %v", err)
	} else {
		githubRepos = repos
		logInfo("Initially found %d trending repositories", len(repos))
	}

	// Research papers
	papersList, err := papers.FetchTopPapers()
	if err != nil {
		logError("Initial papers fetching error: %v", err)
	} else {
		researchPapers = papersList
		logInfo("Initially found %d research papers", len(papersList))
	}

	lastUpdated = time.Now()

	// Setup the web server
	r := gin.Default()

	// 设置信任代理，对于直接暴露到公网的应用，禁用代理信任更安全
	// 如果应用运行在负载均衡器或反向代理后面，请替换为您的代理IP
	r.SetTrustedProxies(nil) // 不信任任何代理，避免IP欺骗

	// Define template functions
	r.SetFuncMap(template.FuncMap{
		"percentMultiply": func(a, b float64) float64 {
			return a * b
		},
		"divScore": func(a, b float64) float64 {
			return a / b
		},
		"floorScore": func(n float64) float64 {
			return math.Floor(n)
		},
		"loopCount": func(n int) []float64 {
			var result []float64
			for i := 0; i < n; i++ {
				result = append(result, float64(i))
			}
			return result
		},
		"subScore": func(a float64, b float64) float64 {
			return a - b
		},
		"gt": func(a, b interface{}) bool {
			// 处理不同类型的比较
			switch v1 := a.(type) {
			case int:
				switch v2 := b.(type) {
				case int:
					return v1 > v2
				case float64:
					return float64(v1) > v2
				}
			case float64:
				switch v2 := b.(type) {
				case int:
					return v1 > float64(v2)
				case float64:
					return v1 > v2
				}
			}
			// 如果类型不匹配或不支持，返回false
			return false
		},
		"eq": func(a, b interface{}) bool {
			// 处理不同类型的比较
			switch v1 := a.(type) {
			case int:
				switch v2 := b.(type) {
				case int:
					return v1 == v2
				case float64:
					return float64(v1) == v2
				}
			case float64:
				switch v2 := b.(type) {
				case int:
					return v1 == float64(v2)
				case float64:
					return v1 == v2
				}
			}
			// 如果类型不匹配或不支持，返回false
			return false
		},
		"lt": func(a, b interface{}) bool {
			// 处理不同类型的比较
			switch v1 := a.(type) {
			case int:
				switch v2 := b.(type) {
				case int:
					return v1 < v2
				case float64:
					return float64(v1) < v2
				}
			case float64:
				switch v2 := b.(type) {
				case int:
					return v1 < float64(v2)
				case float64:
					return v1 < v2
				}
			}
			// 如果类型不匹配或不支持，返回false
			return false
		},
		"float64": func(n int) float64 {
			return float64(n)
		},
		"int": func(f float64) int {
			return int(f)
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen] + "..."
		},
	})

	// Load templates
	r.LoadHTMLGlob("web/templates/*")

	// Static files
	r.Static("/static", "./web/static")

	// Routes
	r.GET("/", func(c *gin.Context) {
		// 设置默认标题
		title := "LLM News - 最新AI/ML开源仓库、研究论文动态"

		// 处理仓库和论文数据
		combinedRepos := mergeRepositories(githubRepos, []models.Repository{})
		sortedRepos := sortRepositories(combinedRepos)

		// 确保论文URL不为空
		papersWithValidURL := make([]models.Paper, len(researchPapers))
		copy(papersWithValidURL, researchPapers)

		for i := range papersWithValidURL {
			// 如果URL为空，设置一个默认值
			if papersWithValidURL[i].URL == "" {
				papersWithValidURL[i].URL = "https://arxiv.org/search/?query=" + url.QueryEscape(papersWithValidURL[i].Title)
			}
		}

		// 准备模板数据
		data := gin.H{
			"title":       title,
			"lastUpdated": lastUpdated.Format("2006-01-02 15:04:05"),
			"now":         time.Now(),
			"repos":       sortedRepos,
			"papers":      papersWithValidURL,
		}

		c.HTML(200, "index.html", data)
	})

	// API endpoints
	r.GET("/api/repos", func(c *gin.Context) {
		combinedRepos := mergeRepositories(githubRepos, []models.Repository{})
		sortedRepos := sortRepositories(combinedRepos)
		c.JSON(200, sortedRepos)
	})

	r.GET("/api/research-articles", func(c *gin.Context) {
		// 确保论文URL不为空
		papersWithValidURL := make([]models.Paper, len(researchPapers))
		copy(papersWithValidURL, researchPapers)

		for i := range papersWithValidURL {
			// 如果URL为空，设置一个默认值
			if papersWithValidURL[i].URL == "" {
				papersWithValidURL[i].URL = "https://arxiv.org/search/?query=" + url.QueryEscape(papersWithValidURL[i].Title)
			}
		}
		c.JSON(200, papersWithValidURL)
	})

	// 为了向后兼容，保留/api/papers接口，但重定向到/api/research-articles
	r.GET("/api/papers", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/api/research-articles")
	})

	r.GET("/api/stats", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"last_updated":          lastUpdated,
			"trending_repos_count":  len(githubRepos),
			"research_papers_count": len(researchPapers),
		})
	})

	// 添加新的API路由用于模型特定仓库搜索
	r.GET("/api/model-repos/:model", searchModelReposHandler)

	// Start the server
	localIP := getLocalIP()
	serverAddr := fmt.Sprintf("%s:8081", localIP)
	logInfo("Starting server on http://%s:8081", localIP)
	if err := r.Run(serverAddr); err != nil {
		logError("Failed to start server: %v", err)
		panic(err) // 服务器启动失败，需要终止程序
	}
}

// mergeRepositories combines repositories from different sources and removes duplicates
func mergeRepositories(repos1, repos2 []models.Repository) []models.Repository {
	// Create a map to detect duplicates
	repoMap := make(map[string]models.Repository)

	// Add all repos from first source
	for _, repo := range repos1 {
		repoMap[repo.Name] = repo
	}

	// Add repos from second source (if not already added)
	for _, repo := range repos2 {
		if _, exists := repoMap[repo.Name]; !exists {
			repoMap[repo.Name] = repo
		}
	}

	// Convert back to slice
	result := make([]models.Repository, 0, len(repoMap))
	for _, repo := range repoMap {
		result = append(result, repo)
	}

	return result
}

// sortRepositories sorts repositories based on their name
func sortRepositories(repos []models.Repository) []models.Repository {
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos
}

// 添加一个直接从GitHub搜索特定模型的API
func searchModelReposHandler(c *gin.Context) {
	modelName := c.Param("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Model name is required"})
		return
	}

	// 定义各个模型的搜索关键词
	searchTerms := map[string][]string{
		"cursor":   {"getcursor", "cursor-ai", "cursor ai", "cursor-editor"},
		"deepseek": {"deepseek-ai", "deepseek coder", "deepseek-coder", "deepseek llm"},
		"hunyuan":  {"tencent hunyuan", "hunyuanvideo", "hunyuandit", "tencent-hunyuan"},
		"claude":   {"anthropic claude", "claude-3", "claude-instant", "anthropic-claude"},
		"gemini":   {"google gemini", "google-gemini", "gemini-pro", "gemini-ultra"},
		"llama":    {"meta-llama", "llama3", "llama-3", "llama-2", "meta llama"},
		"qwen":     {"alibaba qwen", "qwenlm", "qwen-vl", "qwen-7b", "aliyun qwen"},
		"gpt":      {"chatgpt", "gpt-4", "gpt-3.5", "openai gpt", "gpt-turbo"},
		"文心一言":     {"文心一言", "baidu ernie", "wenxin", "百度文心"},
	}

	terms, exists := searchTerms[modelName]
	if !exists {
		terms = []string{modelName} // 如果没有预定义的关键词，使用模型名称本身
	}

	// 构建搜索查询，加上AI关键词确保返回相关结果
	query := strings.Join(terms, " OR ") + " AI language model"

	// 从GitHub直接获取仓库
	repos := directSearchGitHub(query)

	// 对结果进行二次过滤，确保它们与模型相关
	var filteredRepos []models.Repository
	for _, repo := range repos {
		// 检查是否与模型相关
		isRelevant := false
		name := strings.ToLower(repo.Name)
		desc := strings.ToLower(repo.Description)

		// 检查名称或描述中是否包含任何搜索词
		for _, term := range terms {
			if strings.Contains(name, strings.ToLower(term)) ||
				strings.Contains(desc, strings.ToLower(term)) {
				isRelevant = true
				break
			}
		}

		// 如果仓库名就是模型名也算相关
		if strings.Contains(name, strings.ToLower(modelName)) {
			isRelevant = true
		}

		if isRelevant {
			filteredRepos = append(filteredRepos, repo)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"model": modelName,
		"repos": filteredRepos,
	})
}

// 直接从GitHub搜索仓库
func directSearchGitHub(query string) []models.Repository {
	// 构建GitHub API搜索URL
	searchURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc", url.QueryEscape(query))

	// 发送请求到GitHub API
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("Error creating GitHub API request: %v", err)
		return []models.Repository{}
	}

	// 添加GitHub API所需的头信息
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	// 如果有GitHub API令牌，可以添加认证头以增加API速率限制
	githubToken := os.Getenv("GITHUB_API_TOKEN")
	if githubToken != "" {
		req.Header.Add("Authorization", "token "+githubToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching from GitHub API: %v", err)
		return []models.Repository{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("GitHub API returned non-200 status: %d", resp.StatusCode)
		return []models.Repository{}
	}

	// 解析GitHub API响应
	var result struct {
		Items []struct {
			FullName        string `json:"full_name"`
			HTMLURL         string `json:"html_url"`
			Description     string `json:"description"`
			StargazersCount int    `json:"stargazers_count"`
			Language        string `json:"language"`
			UpdatedAt       string `json:"updated_at"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding GitHub API response: %v", err)
		return []models.Repository{}
	}

	// 转换为我们的仓库模型
	repos := make([]models.Repository, 0, len(result.Items))
	for _, item := range result.Items {
		// 解析最后更新时间
		updatedAt, _ := time.Parse(time.RFC3339, item.UpdatedAt)

		repo := models.Repository{
			Name:        item.FullName,
			URL:         item.HTMLURL,
			Description: item.Description,
			Stars:       item.StargazersCount,
			Language:    item.Language,
			LastCommit:  updatedAt,
			TrendMetrics: models.TrendMetrics{
				Stars24h: 0, // 无法从搜索API获取这些数据
				Views7d:  0, // 使用正确的字段名
			},
			GainedStars:    0,   // 无法从搜索API获取这些数据
			RelevanceScore: 4.5, // 默认相关性分数
		}
		repos = append(repos, repo)
	}

	return repos
}
