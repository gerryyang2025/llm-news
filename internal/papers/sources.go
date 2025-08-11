package papers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gerryyang2025/llm-news/internal/models"
)

// FetchOtherBlogPosts 抓取技术博客文章
func FetchOtherBlogPosts() ([]models.Paper, error) {
	var results []models.Paper
	var errors []string

	// 获取HackerNews热门AI文章
	hackerNewsPosts, err := fetchHackerNewsAIArticles()
	if err != nil {
		log.Printf("Warning: Error fetching from HackerNews: %v", err)
		errors = append(errors, fmt.Sprintf("HackerNews: %v", err))
	} else {
		results = append(results, hackerNewsPosts...)
	}

	// 获取Dev.to热门AI文章
	devToPosts, err := fetchDevToAIArticles()
	if err != nil {
		log.Printf("Warning: Error fetching from Dev.to: %v", err)
		errors = append(errors, fmt.Sprintf("Dev.to: %v", err))
	} else {
		results = append(results, devToPosts...)
	}

	// 注释掉机器之心数据源，因为404错误
	/*
		// 获取机器之心热门AI文章
		jiqizhixinPosts, err := fetchJiqizhixinArticles()
		if err != nil {
			log.Printf("Warning: Error fetching from 机器之心: %v", err)
			errors = append(errors, fmt.Sprintf("机器之心: %v", err))
		} else {
			results = append(results, jiqizhixinPosts...)
		}
	*/

	// 获取CSDN热门AI文章
	csdnPosts, err := fetchCSDNArticles()
	if err != nil {
		log.Printf("Warning: Error fetching from CSDN: %v", err)
		errors = append(errors, fmt.Sprintf("CSDN: %v", err))
	} else {
		results = append(results, csdnPosts...)
	}

	// 注释掉InfoQ中文站，因为451错误
	/*
		// 获取InfoQ中文站热门AI文章
		infoqPosts, err := fetchInfoQArticles()
		if err != nil {
			log.Printf("Warning: Error fetching from InfoQ: %v", err)
			errors = append(errors, fmt.Sprintf("InfoQ: %v", err))
		} else {
			results = append(results, infoqPosts...)
		}
	*/

	// 如果所有数据源都获取失败，返回明确的错误
	if len(results) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to fetch articles from all sources: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// 获取HackerNews上热门的AI相关文章
func fetchHackerNewsAIArticles() ([]models.Paper, error) {
	// 获取HackerNews最新故事
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// 获取最新的top stories
	resp, err := client.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HackerNews top stories: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from HackerNews: %d", resp.StatusCode)
	}

	// 解析故事ID
	var storyIDs []int
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("failed to decode HackerNews response: %v", err)
	}

	// 只检查前30个故事
	storyLimit := 30
	if len(storyIDs) > storyLimit {
		storyIDs = storyIDs[:storyLimit]
	}

	// AI相关关键词
	aiKeywords := []string{
		"ai", "artificial intelligence", "machine learning", "ml", "llm", "large language model",
		"chatgpt", "gpt", "claude", "gemini", "openai", "anthropic", "llama", "mistral",
		"huggingface", "neural network", "deep learning", "diffusion", "transformer", "nlp",
	}

	var results []models.Paper

	// 获取每个故事的详情，找出AI相关的
	for _, id := range storyIDs {
		storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
		storyResp, err := client.Get(storyURL)
		if err != nil {
			log.Printf("Warning: Failed to fetch HackerNews story %d: %v", id, err)
			continue
		}

		var story struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Score int    `json:"score"`
			Time  int64  `json:"time"`
			By    string `json:"by"`
			Text  string `json:"text,omitempty"`
		}

		if err := json.NewDecoder(storyResp.Body).Decode(&story); err != nil {
			storyResp.Body.Close()
			log.Printf("Warning: Failed to decode HackerNews story %d: %v", id, err)
			continue
		}
		storyResp.Body.Close()

		// 检查是否与AI相关
		isAIRelated := false
		storyTitle := strings.ToLower(story.Title)
		for _, keyword := range aiKeywords {
			if strings.Contains(storyTitle, strings.ToLower(keyword)) {
				isAIRelated = true
				break
			}
		}

		if isAIRelated {
			paper := models.Paper{
				Title:            story.Title,
				URL:              story.URL,
				Authors:          []string{story.By},
				PublishedDate:    time.Unix(story.Time, 0),
				Source:           "HackerNews",
				Summary:          story.Text,
				Keywords:         extractKeywords(story.Title + " " + story.Text),
				CitationCount:    story.Score, // 使用得分作为引用计数
				CitationVelocity: float64(story.Score) / float64(maxInt(1, int(time.Since(time.Unix(story.Time, 0)).Hours()/24))),
				NoveltyScore:     calculateNoveltyScore(story.Title, story.Text),
			}
			results = append(results, paper)

			// 最多只返回5篇AI相关文章
			if len(results) >= 5 {
				break
			}
		}
	}

	return results, nil
}

// 从Dev.to获取热门AI文章
func fetchDevToAIArticles() ([]models.Paper, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// 获取Dev.to上带有AI标签的热门文章
	resp, err := client.Get("https://dev.to/api/articles?tag=ai&top=5")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Dev.to articles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from Dev.to: %d", resp.StatusCode)
	}

	// Dev.to API的返回格式有变化，修改为使用更灵活的解析方式
	var articlesRaw []map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&articlesRaw); err != nil {
		return nil, fmt.Errorf("failed to decode Dev.to response: %v", err)
	}

	var results []models.Paper
	for _, articleRaw := range articlesRaw {
		// 提取标题和URL
		title, _ := articleRaw["title"].(string)
		url, _ := articleRaw["url"].(string)
		publishedAtStr, _ := articleRaw["published_at"].(string)
		description, _ := articleRaw["description"].(string)
		reactionsCount, _ := articleRaw["positive_reactions_count"].(float64)
		readingTime, _ := articleRaw["reading_time_minutes"].(float64)

		// 提取用户名
		var authorName string
		if user, ok := articleRaw["user"].(map[string]interface{}); ok {
			authorName, _ = user["name"].(string)
		}

		// 提取标签 - 处理可能是字符串或字符串数组的情况
		var tags []string
		if tagsRaw, ok := articleRaw["tags"].([]interface{}); ok {
			for _, tag := range tagsRaw {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
		} else if tagsStr, ok := articleRaw["tags"].(string); ok {
			// 如果是逗号分隔的字符串
			tags = strings.Split(tagsStr, ", ")
		}

		publishedDate, _ := time.Parse(time.RFC3339, publishedAtStr)

		paper := models.Paper{
			Title:            title,
			URL:              url,
			Authors:          []string{authorName},
			PublishedDate:    publishedDate,
			Source:           "Dev.to",
			Summary:          description,
			Keywords:         tags,
			CitationCount:    int(reactionsCount),
			CitationVelocity: float64(int(reactionsCount)) / float64(maxInt(1, int(time.Since(publishedDate).Hours()/24))),
			NoveltyScore:     3.5 + float64(minInt(int(readingTime), 30))/10.0, // 基于阅读时间的新颖性评分
		}

		results = append(results, paper)
	}

	return results, nil
}

// 从机器之心获取热门AI文章
func fetchJiqizhixinArticles() ([]models.Paper, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// 机器之心没有公开API，我们需要抓取网页内容
	// 这里使用RSS feed替代，或者直接解析HTML页面
	resp, err := client.Get("https://www.jiqizhixin.com/categories/technical")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch 机器之心 articles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from 机器之心: %d", resp.StatusCode)
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read 机器之心 response: %v", err)
	}

	body := string(bodyBytes)

	// 使用正则表达式提取文章信息
	titleRegex := regexp.MustCompile(`<h4 class="article-item__title">\s*<a[^>]*>([^<]+)</a>`)
	linkRegex := regexp.MustCompile(`<h4 class="article-item__title">\s*<a href="([^"]+)"`)
	dateRegex := regexp.MustCompile(`<span class="article-item__date">([^<]+)</span>`)

	titles := titleRegex.FindAllStringSubmatch(body, -1)
	links := linkRegex.FindAllStringSubmatch(body, -1)
	dates := dateRegex.FindAllStringSubmatch(body, -1)

	var results []models.Paper

	// 限制获取的文章数量
	maxArticles := 5
	if len(titles) > maxArticles {
		titles = titles[:maxArticles]
	}
	if len(links) > maxArticles {
		links = links[:maxArticles]
	}

	for i := 0; i < len(titles) && i < len(links) && len(results) < maxArticles; i++ {
		if len(titles[i]) > 1 && len(links[i]) > 1 {
			title := strings.TrimSpace(titles[i][1])
			link := "https://www.jiqizhixin.com" + links[i][1]

			// 只获取AI相关文章
			if isAIRelated(title) {
				publishedDate := time.Now() // 如果无法解析日期，使用当前时间
				if i < len(dates) && len(dates[i]) > 1 {
					// 尝试解析日期，格式可能是"2023-01-01"或类似格式
					if parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(dates[i][1])); err == nil {
						publishedDate = parsedDate
					}
				}

				paper := models.Paper{
					Title:            title,
					URL:              link,
					Authors:          []string{"机器之心"},
					PublishedDate:    publishedDate,
					Source:           "机器之心",
					Summary:          fmt.Sprintf("来自机器之心的AI技术文章：%s", title),
					Keywords:         extractKeywords(title),
					CitationCount:    10, // 假设的引用计数
					CitationVelocity: 1.0,
					NoveltyScore:     calculateNoveltyScore(title, ""),
				}
				results = append(results, paper)
			}
		}
	}

	return results, nil
}

// 从CSDN获取热门AI文章
func fetchCSDNArticles() ([]models.Paper, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// CSDN AI专区
	resp, err := client.Get("https://blog.csdn.net/nav/ai")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CSDN articles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from CSDN: %d", resp.StatusCode)
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSDN response: %v", err)
	}

	body := string(bodyBytes)

	// 使用正则表达式提取文章信息
	titleRegex := regexp.MustCompile(`<a class="title" href="[^"]+">([^<]+)</a>`)
	linkRegex := regexp.MustCompile(`<a class="title" href="([^"]+)"`)

	titles := titleRegex.FindAllStringSubmatch(body, -1)
	links := linkRegex.FindAllStringSubmatch(body, -1)

	var results []models.Paper

	// 限制获取的文章数量
	maxArticles := 5
	if len(titles) > maxArticles {
		titles = titles[:maxArticles]
	}
	if len(links) > maxArticles {
		links = links[:maxArticles]
	}

	for i := 0; i < len(titles) && i < len(links) && len(results) < maxArticles; i++ {
		if len(titles[i]) > 1 && len(links[i]) > 1 {
			title := strings.TrimSpace(titles[i][1])
			link := links[i][1]

			// 只获取AI相关文章
			if isAIRelated(title) {
				paper := models.Paper{
					Title:            title,
					URL:              link,
					Authors:          []string{"CSDN博客"},
					PublishedDate:    time.Now(), // 假设为当前时间
					Source:           "CSDN",
					Summary:          fmt.Sprintf("来自CSDN的AI技术文章：%s", title),
					Keywords:         extractKeywords(title),
					CitationCount:    5, // 假设的引用计数
					CitationVelocity: 0.5,
					NoveltyScore:     calculateNoveltyScore(title, ""),
				}
				results = append(results, paper)
			}
		}
	}

	return results, nil
}

// 从InfoQ中文站获取热门AI文章
func fetchInfoQArticles() ([]models.Paper, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	// InfoQ AI专区
	resp, err := client.Get("https://www.infoq.cn/topic/AI")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch InfoQ articles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from InfoQ: %d", resp.StatusCode)
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read InfoQ response: %v", err)
	}

	body := string(bodyBytes)

	// 使用正则表达式提取文章信息
	titleRegex := regexp.MustCompile(`<div class="article-item__title[^"]*">([^<]+)</div>`)
	linkRegex := regexp.MustCompile(`<a href="(/[^"]+)" target="_blank" class="article-item__link">`)
	authorRegex := regexp.MustCompile(`<div class="article-item__author[^"]*">([^<]+)</div>`)

	titles := titleRegex.FindAllStringSubmatch(body, -1)
	links := linkRegex.FindAllStringSubmatch(body, -1)
	authors := authorRegex.FindAllStringSubmatch(body, -1)

	var results []models.Paper

	// 限制获取的文章数量
	maxArticles := 5
	if len(titles) > maxArticles {
		titles = titles[:maxArticles]
	}
	if len(links) > maxArticles {
		links = links[:maxArticles]
	}

	for i := 0; i < len(titles) && i < len(links) && len(results) < maxArticles; i++ {
		if len(titles[i]) > 1 && len(links[i]) > 1 {
			title := strings.TrimSpace(titles[i][1])
			link := "https://www.infoq.cn" + links[i][1]

			var author string
			if i < len(authors) && len(authors[i]) > 1 {
				author = strings.TrimSpace(authors[i][1])
			} else {
				author = "InfoQ作者"
			}

			paper := models.Paper{
				Title:            title,
				URL:              link,
				Authors:          []string{author},
				PublishedDate:    time.Now(), // 假设为当前时间
				Source:           "InfoQ",
				Summary:          fmt.Sprintf("来自InfoQ的AI技术文章：%s", title),
				Keywords:         extractKeywords(title),
				CitationCount:    8, // 假设的引用计数
				CitationVelocity: 0.8,
				NoveltyScore:     calculateNoveltyScore(title, ""),
			}
			results = append(results, paper)
		}
	}

	return results, nil
}

// 从标题和文本中提取关键词
func extractKeywords(text string) []string {
	// 简单的关键词提取实现
	keywords := make(map[string]bool)

	// AI相关关键词
	aiTerms := []string{
		"ai", "artificial intelligence", "machine learning", "ml", "llm", "large language model",
		"chatgpt", "gpt", "claude", "gemini", "openai", "anthropic", "llama", "mistral",
		"huggingface", "neural", "deep learning", "diffusion", "transformer", "nlp",
		"rag", "agents", "multimodal", "vision", "speech", "bert", "rlhf", "fine-tuning",
	}

	text = strings.ToLower(text)
	for _, term := range aiTerms {
		if strings.Contains(text, term) {
			keywords[term] = true
		}
	}

	var result []string
	for keyword := range keywords {
		result = append(result, keyword)
	}

	return result
}

// 计算基于内容的新颖性分数
func calculateNoveltyScore(title, text string) float64 {
	combined := strings.ToLower(title + " " + text)

	// 新技术关键词，这些通常表示更新的内容
	newTechTerms := []string{
		"gpt-4", "claude 3", "gemini", "llama 3", "mistral", "mixtral",
		"multimodal", "vision", "audio", "video", "agents", "rag", "sora",
		"generative", "diffusion", "mamba", "state space model", "tokens",
	}

	score := 3.0 // 基础分数

	// 添加新技术术语加分
	for _, term := range newTechTerms {
		if strings.Contains(combined, term) {
			score += 0.1
		}
	}

	// 限制分数范围
	if score < 1.0 {
		score = 1.0
	} else if score > 5.0 {
		score = 5.0
	}

	return score
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 辅助函数
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// 检查内容是否与AI相关
func isAIRelated(text string) bool {
	text = strings.ToLower(text)

	aiTerms := []string{
		"ai", "machine learning", "ml", "llm", "language model",
		"chatgpt", "gpt", "claude", "gemini", "openai", "anthropic",
		"huggingface", "neural network", "deep learning", "diffusion",
		"transformer", "nlp", "bert", "rlhf", "fine-tuning", "rag", "agent",
	}

	for _, term := range aiTerms {
		if strings.Contains(text, term) {
			return true
		}
	}

	return false
}
