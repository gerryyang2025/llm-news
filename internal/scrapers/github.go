package scrapers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gerryyang2025/llm-news/internal/models"
)

// ScrapeGithubTrending scrapes the GitHub trending page and returns repositories
// filtered by AI-related keywords
func ScrapeGithubTrending() ([]models.Repository, error) {
	// Get repositories from GitHub trending
	repos, err := scrapeBasicTrendingInfo()
	if err != nil {
		return nil, err
	}

	// Filter repositories by AI-related keywords
	aiRepos := filterReposByKeywords(repos, models.AIKeywords)

	// Enrich repositories with additional information
	for i := range aiRepos {
		enrichRepositoryDetails(&aiRepos[i])
	}

	// Apply filter criteria
	filteredRepos := applyFilterCriteria(aiRepos, models.DefaultFilterCriteria())

	// Calculate relevance scores
	calculateRelevanceScores(filteredRepos)

	return filteredRepos, nil
}

// scrapeBasicTrendingInfo scrapes basic information from GitHub trending page
func scrapeBasicTrendingInfo() ([]models.Repository, error) {
	// GitHub trending URLs - we'll get daily, weekly and monthly trending
	urls := []string{
		"https://github.com/trending",                  // Daily trending
		"https://github.com/trending?since=weekly",     // Weekly trending
		"https://github.com/trending?since=monthly",    // Monthly trending
		"https://github.com/trending/python",           // Python trending
		"https://github.com/trending/javascript",       // JavaScript trending
		"https://github.com/trending/typescript",       // TypeScript trending
		"https://github.com/trending/jupyter-notebook", // Jupyter Notebook trending
		"https://github.com/trending/cpp",              // C++ trending
		"https://github.com/trending/go",               // GoLang trending
	}

	allRepos := []models.Repository{}

	// Make HTTP requests to both URLs
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Process each URL
	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Warning: Failed to fetch %s: %v", url, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Warning: Unexpected status code from %s: %d", url, resp.StatusCode)
			resp.Body.Close()
			continue
		}

		// Parse HTML
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Printf("Warning: Failed to parse HTML from %s: %v", url, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Determine if this is daily, weekly or monthly trending
		isWeekly := strings.Contains(url, "weekly")
		isMonthly := strings.Contains(url, "monthly")

		// Iterate over repository items
		doc.Find("article.Box-row").Each(func(i int, s *goquery.Selection) {
			repo := models.Repository{}

			// Get repository name
			nameElem := s.Find("h2 a")
			pathParts := strings.Split(strings.TrimSpace(nameElem.Text()), "/")

			if len(pathParts) >= 2 {
				owner := strings.TrimSpace(pathParts[0])
				repoName := strings.TrimSpace(pathParts[1])
				repo.Name = fmt.Sprintf("%s/%s", owner, repoName)
				repo.URL = fmt.Sprintf("https://github.com/%s", repo.Name)
			} else {
				return // Skip this repository if we can't parse the name
			}

			// Get repository description
			repo.Description = strings.TrimSpace(s.Find("p").Text())

			// Get repository language
			repo.Language = strings.TrimSpace(s.Find("span[itemprop='programmingLanguage']").Text())

			// Get stars count
			starsText := strings.TrimSpace(s.Find("a.Link--muted[href$='stargazers']").Text())
			starsRegex := regexp.MustCompile(`[\d,]+`)
			starsStr := starsRegex.FindString(starsText)
			starsStr = strings.ReplaceAll(starsStr, ",", "")
			if stars, err := strconv.Atoi(starsStr); err == nil {
				repo.Stars = stars
			}

			// Get forks count
			forksText := strings.TrimSpace(s.Find("a.Link--muted[href$='forks']").Text())
			forksRegex := regexp.MustCompile(`[\d,]+`)
			forksStr := forksRegex.FindString(forksText)
			forksStr = strings.ReplaceAll(forksStr, ",", "")
			if forks, err := strconv.Atoi(forksStr); err == nil {
				repo.Forks = forks
			}

			// Get stars gained
			gainedText := strings.TrimSpace(s.Find("span.d-inline-block.float-sm-right").Text())
			gainedRegex := regexp.MustCompile(`[\d,]+`)
			gainedStr := gainedRegex.FindString(gainedText)
			gainedStr = strings.ReplaceAll(gainedStr, ",", "")
			if gained, err := strconv.Atoi(gainedStr); err == nil {
				repo.GainedStars = gained
				// Set stars/forks in the last 24h based on the timeframe
				if isMonthly {
					// Average daily gain for monthly trending
					repo.TrendMetrics.Stars24h = gained / 30
				} else if isWeekly {
					// Average daily gain for weekly trending
					repo.TrendMetrics.Stars24h = gained / 7
				} else {
					repo.TrendMetrics.Stars24h = gained
				}
			}

			repo.LastUpdated = time.Now()
			repo.RelevanceScore = 0.5 // Default mid-level score

			// Skip duplicate repositories
			for _, existingRepo := range allRepos {
				if existingRepo.Name == repo.Name {
					return
				}
			}

			allRepos = append(allRepos, repo)
		})
	}

	// 尝试补充额外的仓库，如果当前数量不足50个
	if len(allRepos) < 50 {
		additionalRepos, err := fetchAdditionalRepos(50 - len(allRepos))
		if err == nil && len(additionalRepos) > 0 {
			for _, repo := range additionalRepos {
				// 检查是否存在重复
				isDuplicate := false
				for _, existingRepo := range allRepos {
					if existingRepo.Name == repo.Name {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					allRepos = append(allRepos, repo)
				}
			}
		}
	}

	if len(allRepos) == 0 {
		return nil, errors.New("no repositories found, the scraper might need to be updated")
	}

	return allRepos, nil
}

// fetchAdditionalRepos fetches additional repositories using GitHub API search
func fetchAdditionalRepos(count int) ([]models.Repository, error) {
	if count <= 0 {
		return []models.Repository{}, nil
	}

	// 使用GitHub API搜索相关仓库
	searchQueries := []string{
		"topic:artificial-intelligence sort:stars",
		"topic:ai sort:stars",
		"topic:machine-learning sort:stars",
		"topic:deep-learning sort:stars",
		"topic:llm sort:stars",
		"topic:nlp sort:stars",
		"topic:language-model sort:stars",
		"topic:diffusion-models sort:stars",
		// 添加C++相关查询
		"language:cpp topic:ai sort:stars",
		"language:cpp topic:machine-learning sort:stars",
		"language:cpp topic:neural-network sort:stars",
		"language:cpp topic:deep-learning sort:stars",
		// 添加GoLang相关查询
		"language:go topic:ai sort:stars",
		"language:go topic:machine-learning sort:stars",
		"language:go topic:llm sort:stars",
		"language:go topic:rag sort:stars",
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	additionalRepos := []models.Repository{}

	// 每个查询获取一定数量，直到达到目标数量
	perQueryCount := (count / len(searchQueries)) + 1

	for _, query := range searchQueries {
		if len(additionalRepos) >= count {
			break
		}

		// 构建GitHub搜索API URL
		url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=%d",
			strings.ReplaceAll(query, " ", "+"), perQueryCount)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}

		// 添加User-Agent头以避免GitHub API限制
		req.Header.Add("User-Agent", "LLM-News-Agent")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// 解析JSON响应
		var searchResult struct {
			Items []struct {
				Name            string   `json:"name"`
				FullName        string   `json:"full_name"`
				HTMLURL         string   `json:"html_url"`
				Description     string   `json:"description"`
				StargazersCount int      `json:"stargazers_count"`
				ForksCount      int      `json:"forks_count"`
				Language        string   `json:"language"`
				Topics          []string `json:"topics"`
				UpdatedAt       string   `json:"updated_at"`
				PushedAt        string   `json:"pushed_at"`
			} `json:"items"`
		}

		if err := json.Unmarshal(body, &searchResult); err != nil {
			continue
		}

		// 处理搜索结果
		for _, item := range searchResult.Items {
			// 创建仓库对象
			repo := models.Repository{
				Name:        item.FullName,
				URL:         item.HTMLURL,
				Description: item.Description,
				Language:    item.Language,
				Stars:       item.StargazersCount,
				Forks:       item.ForksCount,
				LastUpdated: time.Now(),
				TechStack:   item.Topics,
				TrendMetrics: models.TrendMetrics{
					// 估算星星增长数
					Stars24h: item.StargazersCount / 1000, // 粗略估计每天获得的星星数
				},
				RelevanceScore: 0.5, // 默认中等分数
			}

			// 解析提交日期
			if item.PushedAt != "" {
				if t, err := time.Parse(time.RFC3339, item.PushedAt); err == nil {
					repo.LastCommit = t
				}
			}

			additionalRepos = append(additionalRepos, repo)

			// 如果达到目标数量，则停止
			if len(additionalRepos) >= count {
				break
			}
		}
	}

	return additionalRepos, nil
}

// enrichRepositoryDetails adds additional information to a repository using GitHub API
func enrichRepositoryDetails(repo *models.Repository) {
	// Extract owner and repo name
	parts := strings.Split(repo.Name, "/")
	if len(parts) != 2 {
		return
	}

	owner := parts[0]
	repoName := parts[1]

	// GitHub API URL
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repoName)

	// Make HTTP request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	// Add User-Agent header to avoid GitHub API limitations
	req.Header.Add("User-Agent", "LLM-News-Agent")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Parse JSON response
	var githubRepo struct {
		Description     string   `json:"description"`
		Language        string   `json:"language"`
		StargazersCount int      `json:"stargazers_count"`
		ForksCount      int      `json:"forks_count"`
		UpdatedAt       string   `json:"updated_at"`
		PushedAt        string   `json:"pushed_at"`
		Topics          []string `json:"topics"`
		HasPages        bool     `json:"has_pages"`
		HasWiki         bool     `json:"has_wiki"`
		HasIssues       bool     `json:"has_issues"`
	}

	if err := json.Unmarshal(body, &githubRepo); err != nil {
		return
	}

	// Update repository with GitHub details
	if githubRepo.Description != "" {
		repo.Description = githubRepo.Description
	}
	if githubRepo.Language != "" {
		repo.Language = githubRepo.Language
	}
	if githubRepo.StargazersCount > 0 {
		// If API stars are different, update but preserve the gained stars
		// This will provide more accurate information
		if repo.Stars != githubRepo.StargazersCount {
			repo.Stars = githubRepo.StargazersCount
		}
	}
	repo.Forks = githubRepo.ForksCount

	// Parse dates
	if githubRepo.PushedAt != "" {
		if t, err := time.Parse(time.RFC3339, githubRepo.PushedAt); err == nil {
			repo.LastCommit = t
		}
	}

	// Set tech stack from topics
	if len(githubRepo.Topics) > 0 {
		repo.TechStack = githubRepo.Topics
	} else {
		// If no topics are available, use the language as the tech stack
		if repo.Language != "" {
			repo.TechStack = []string{repo.Language}
		}
	}

	// Check if it has docs
	repo.HasDocs = githubRepo.HasWiki || githubRepo.HasPages
	repo.HasWiki = githubRepo.HasWiki

	// 设置文档URL
	if githubRepo.HasWiki {
		repo.DocsURL = fmt.Sprintf("https://github.com/%s/wiki", repo.Name)
	}

	// Check if README exists
	readmeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repoName)
	readmeReq, err := http.NewRequest("GET", readmeURL, nil)
	if err == nil {
		readmeReq.Header.Add("User-Agent", "LLM-News-Agent")
		readmeResp, err := client.Do(readmeReq)
		if err == nil && readmeResp.StatusCode == http.StatusOK {
			repo.HasDocs = true
			repo.HasReadme = true
			if repo.DocsURL == "" {
				repo.DocsURL = fmt.Sprintf("https://github.com/%s#readme", repo.Name)
			}
			readmeResp.Body.Close()
		}
	}

	// Calculate forks gained
	// For simplicity, we'll estimate this based on the stars gained
	// In a real implementation, you'd track this over time
	if repo.GainedStars > 0 {
		ratio := float64(repo.Forks) / float64(repo.Stars)
		repo.GainedForks = int(float64(repo.GainedStars) * ratio)
		repo.TrendMetrics.Forks24h = repo.GainedForks
	}

	// 计算并获取模型分类
	repo.GetModelCategories()
}

// filterReposByKeywords filters repositories by checking if their name or description
// contains any of the given keywords
func filterReposByKeywords(repos []models.Repository, keywords []string) []models.Repository {
	filtered := []models.Repository{}

	// 添加更多可能相关的仓库
	potentialRepos := []models.Repository{}

	for _, repo := range repos {
		lowerName := strings.ToLower(repo.Name)
		lowerDesc := strings.ToLower(repo.Description)

		// 强匹配: 名称或描述中直接包含核心关键词
		coreKeywords := []string{"llm", "ai", "ml", "gpt", "bert", "nlp", "language-model", "machine-learning", "deep-learning"}

		// 检查核心关键词
		foundCore := false
		for _, keyword := range coreKeywords {
			if strings.Contains(lowerName, keyword) || strings.Contains(lowerDesc, keyword) {
				filtered = append(filtered, repo)
				log.Printf("Found AI repository: %s", repo.Name)
				foundCore = true
				break
			}
		}

		if foundCore {
			continue // 已经添加过，跳过后续检查
		}

		// 弱匹配: 检查所有关键词
		for _, keyword := range keywords {
			if strings.Contains(lowerName, keyword) || strings.Contains(lowerDesc, keyword) {
				potentialRepos = append(potentialRepos, repo)
				break
			}
		}
	}

	// 将可能相关的仓库添加到结果中
	for _, repo := range potentialRepos {
		filtered = append(filtered, repo)
		log.Printf("Found AI repository: %s", repo.Name)
	}

	return filtered
}

// applyFilterCriteria filters repositories based on the specified criteria
func applyFilterCriteria(repos []models.Repository, criteria models.FilterCriteria) []models.Repository {
	// 如果仓库数量少于50个，则跳过过滤直接返回
	if len(repos) < 50 {
		return repos
	}

	filtered := []models.Repository{}

	for _, repo := range repos {
		// Check minimum stars growth rate
		if repo.TrendMetrics.Stars24h < criteria.MinStarsGrowthRate {
			continue
		}

		// Check maximum days since last commit
		if !repo.LastCommit.IsZero() {
			daysSinceLastCommit := time.Since(repo.LastCommit).Hours() / 24
			if daysSinceLastCommit > float64(criteria.MaxDaysSinceCommit) {
				continue
			}
		}

		// Check documentation requirement
		if criteria.RequiresDocumentation && !repo.HasDocs {
			continue
		}

		// Check minimum relevance score
		if repo.RelevanceScore < criteria.MinRelevanceScore {
			continue
		}

		filtered = append(filtered, repo)
	}

	// 如果过滤后数量不足，则适当降低标准，保留最相关的仓库
	if len(filtered) < 50 && len(repos) > 0 {
		// 按相关性分数排序
		sort.Slice(repos, func(i, j int) bool {
			return repos[i].RelevanceScore > repos[j].RelevanceScore
		})

		// 至少返回50个仓库或所有仓库（如果总数少于50）
		maxReturn := 50
		if len(repos) < maxReturn {
			maxReturn = len(repos)
		}

		return repos[:maxReturn]
	}

	return filtered
}

// calculateRelevanceScores calculates relevance scores for repositories
func calculateRelevanceScores(repos []models.Repository) {
	for i := range repos {
		// Calculate base score based on stars and engagement
		starsScore := minFloat(float64(repos[i].Stars)/5000.0, 1.0) * 0.25                // 降低星星权重
		growthScore := minFloat(float64(repos[i].TrendMetrics.Stars24h)/50.0, 1.0) * 0.35 // 降低增长率权重
		// Calculate recency score
		recencyScore := 0.0
		if !repos[i].LastCommit.IsZero() {
			daysSinceLastCommit := time.Since(repos[i].LastCommit).Hours() / 24
			recencyScore = (1.0 - minFloat(daysSinceLastCommit/30.0, 1.0)) * 0.15 // 使用30天作为时间窗口
		}

		// Calculate keyword relevance score
		keywordScore := 0.25 // 提高关键词基础分
		relevantKeywords := []string{
			"llm", "agent", "multimodal", "rlhf", "diffusion", "agi", "ai", "ml",
			"gpt", "bert", "transformer", "nlp", "language-model", "claude", "gemini",
			"fine-tuning", "prompt", "rag", "anthropic", "openai", "text-to-image",
		}

		// 在名称和描述中查找关键词
		lowerName := strings.ToLower(repos[i].Name)
		lowerDesc := strings.ToLower(repos[i].Description)

		for _, keyword := range relevantKeywords {
			if strings.Contains(lowerName, keyword) {
				keywordScore += 0.03 // 名称匹配给更高权重
			}
			if strings.Contains(lowerDesc, keyword) {
				keywordScore += 0.01 // 描述匹配给较低权重
			}
		}

		// 检查技术栈中的关键词
		for _, tech := range repos[i].TechStack {
			techLower := strings.ToLower(tech)
			for _, keyword := range relevantKeywords {
				if strings.Contains(techLower, keyword) {
					keywordScore += 0.02 // 技术栈匹配
					break
				}
			}
		}

		keywordScore = minFloat(keywordScore, 0.35) // 限制关键词分数上限

		// Sum up for final score
		repos[i].RelevanceScore = starsScore + growthScore + recencyScore + keywordScore

		// Ensure the score is between 0 and 1
		repos[i].RelevanceScore = minFloat(repos[i].RelevanceScore, 1.0)
	}
}

// minFloat returns the minimum of two float64 values
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
