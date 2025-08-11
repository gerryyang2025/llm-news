package scrapers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gerryyang2025/llm-news/internal/models"
)

// PapersWithCodeRepository represents a repository from Papers with Code
type PapersWithCodeRepository struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Stars       int      `json:"stars"`
	Framework   string   `json:"framework"`
	Tasks       []string `json:"tasks"`
	PaperURL    string   `json:"paper_url"`
	PaperTitle  string   `json:"paper_title"`
}

// ScrapePapersWithCode scrapes the Papers with Code trending repositories
func ScrapePapersWithCode() ([]models.Repository, error) {
	// 存储所有获取的论文仓库
	allRepos := []models.Repository{}

	// 尝试从Papers with Code获取数据
	papersWithCodeRepos, err := scrapePapersWithCodeAPI()
	if err != nil {
		fmt.Printf("Warning: Failed to fetch from Papers with Code API: %v\n", err)
	} else {
		allRepos = append(allRepos, papersWithCodeRepos...)
	}

	// 尝试从GitHub专题列表获取AI论文实现
	githubAIPapersRepos, err := scrapeGitHubAIPapers()
	if err != nil {
		fmt.Printf("Warning: Failed to fetch from GitHub AI Papers: %v\n", err)
	} else {
		allRepos = append(allRepos, githubAIPapersRepos...)
	}

	return allRepos, nil
}

// scrapePapersWithCodeAPI 从Papers with Code API获取数据
func scrapePapersWithCodeAPI() ([]models.Repository, error) {
	// Papers with Code API endpoint
	// 使用较广泛的主题并增加结果数
	url := "https://paperswithcode.com/api/v1/papers/?topics=language-modelling,transformer,nlp,llm,gpt,diffusion-models,computer-vision,retrieval,optimization&limit=50&page=1"

	// Make HTTP request
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 添加用户代理以避免被阻止
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Papers with Code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from Papers with Code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var apiResponse struct {
		Results []struct {
			Title        string          `json:"title"`
			URL          string          `json:"url"`
			PublishedAt  time.Time       `json:"published_at"`
			Authors      json.RawMessage `json:"authors"`
			Abstract     string          `json:"abstract"`
			Repositories []struct {
				URL       string `json:"url"`
				Framework string `json:"framework"`
				Stars     int    `json:"stars"`
			} `json:"repositories"`
			Tasks []struct {
				Name string `json:"name"`
			} `json:"tasks"`
		} `json:"results"`
		Count int `json:"count"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Papers with Code JSON: %w", err)
	}

	repos := []models.Repository{}

	// Process each paper with its repositories
	for _, paper := range apiResponse.Results {
		// Skip papers without repositories
		if len(paper.Repositories) == 0 {
			continue
		}

		// Get tasks
		tasks := []string{}
		for _, task := range paper.Tasks {
			tasks = append(tasks, task.Name)
		}

		// 灵活处理作者字段
		authors := []string{}

		// 尝试解析为数组形式
		var authorsArray []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(paper.Authors, &authorsArray); err == nil {
			for _, author := range authorsArray {
				authors = append(authors, author.Name)
			}
		} else {
			// 尝试解析为单个字符串
			var authorStr string
			if err := json.Unmarshal(paper.Authors, &authorStr); err == nil {
				authors = append(authors, authorStr)
			} else {
				// 尝试解析为字符串数组
				var authorStrArray []string
				if err := json.Unmarshal(paper.Authors, &authorStrArray); err == nil {
					authors = append(authors, authorStrArray...)
				} else {
					// 使用默认作者
					authors = append(authors, "Unknown Author")
				}
			}
		}

		// Process each repository from this paper
		for _, repo := range paper.Repositories {
			// Extract repository name from URL
			repoNameMatch := regexp.MustCompile(`github\.com/([^/]+/[^/]+)`).FindStringSubmatch(repo.URL)
			if len(repoNameMatch) < 2 {
				continue
			}
			repoName := repoNameMatch[1]

			// Check if this is a fork (skip if it is)
			if strings.Contains(repo.URL, "/fork") {
				continue
			}

			// Create new repository entry
			repository := models.Repository{
				Name:        repoName,
				URL:         repo.URL,
				Description: truncateString(paper.Abstract, 200),
				Language:    repo.Framework,
				Stars:       repo.Stars,
				GainedStars: 0, // We don't know the daily gain from the API
				LastUpdated: time.Now(),
				LastCommit:  paper.PublishedAt, // Using paper publish date as a proxy
				TechStack:   []string{repo.Framework},
				TrendMetrics: models.TrendMetrics{
					Stars24h: 0, // We'll estimate this later
				},
				RelevanceScore: 0.8,  // Default high score for papers with code
				HasDocs:        true, // Assume it has docs since it's from papers with code
				Source:         "Papers with Code",
				PaperURL:       paper.URL,
				PaperTitle:     paper.Title,
				Authors:        authors,
			}

			// Try to fetch additional repository details from GitHub
			enrichRepositoryWithGitHubDetails(&repository)

			repos = append(repos, repository)
		}
	}

	return repos, nil
}

// scrapeGitHubAIPapers 从GitHub获取AI论文实现
func scrapeGitHubAIPapers() ([]models.Repository, error) {
	// 定义一些知名的AI论文实现仓库
	knownRepos := []struct {
		Owner       string
		Repo        string
		Description string
		PaperTitle  string
		PaperURL    string
	}{
		{
			Owner:       "lucidrains",
			Repo:        "DALLE2-pytorch",
			Description: "Implementation of DALL-E 2, OpenAI's updated text-to-image synthesis neural network, in PyTorch",
			PaperTitle:  "Hierarchical Text-Conditional Image Generation with CLIP Latents",
			PaperURL:    "https://arxiv.org/abs/2204.06125",
		},
		{
			Owner:       "facebookresearch",
			Repo:        "llama",
			Description: "Inference code for LLaMA models",
			PaperTitle:  "LLaMA: Open and Efficient Foundation Language Models",
			PaperURL:    "https://arxiv.org/abs/2302.13971",
		},
		{
			Owner:       "jina-ai",
			Repo:        "clip-as-service",
			Description: "Embed images and sentences into fixed-length vectors with CLIP",
			PaperTitle:  "Learning Transferable Visual Models From Natural Language Supervision",
			PaperURL:    "https://arxiv.org/abs/2103.00020",
		},
		{
			Owner:       "huggingface",
			Repo:        "diffusers",
			Description: "Diffusers: State-of-the-art diffusion models for image and audio generation in PyTorch",
			PaperTitle:  "High-Resolution Image Synthesis with Latent Diffusion Models",
			PaperURL:    "https://arxiv.org/abs/2112.10752",
		},
		{
			Owner:       "Lightning-AI",
			Repo:        "lit-llama",
			Description: "Implementation of the LLaMA language model based on nanoGPT. Supports QLoRA, LoRA, LLaMA-Adapter, and more",
			PaperTitle:  "LLaMA: Open and Efficient Foundation Language Models",
			PaperURL:    "https://arxiv.org/abs/2302.13971",
		},
		{
			Owner:       "salesforce",
			Repo:        "BLIP",
			Description: "PyTorch implementation of BLIP: Bootstrapping Language-Image Pre-training for Unified Vision-Language Understanding and Generation",
			PaperTitle:  "BLIP: Bootstrapping Language-Image Pre-training for Unified Vision-Language Understanding and Generation",
			PaperURL:    "https://arxiv.org/abs/2201.12086",
		},
		{
			Owner:       "microsoft",
			Repo:        "LoRA",
			Description: "Code for loralib, an implementation of 'LoRA: Low-Rank Adaptation of Large Language Models'",
			PaperTitle:  "LoRA: Low-Rank Adaptation of Large Language Models",
			PaperURL:    "https://arxiv.org/abs/2106.09685",
		},
		{
			Owner:       "chroma-core",
			Repo:        "chroma",
			Description: "The AI-native open-source embedding database",
			PaperTitle:  "Chroma: The AI-native open-source embedding database",
			PaperURL:    "https://www.trychroma.com/",
		},
		{
			Owner:       "ggerganov",
			Repo:        "llama.cpp",
			Description: "Port of Facebook's LLaMA model in C/C++",
			PaperTitle:  "LLaMA: Open and Efficient Foundation Language Models",
			PaperURL:    "https://arxiv.org/abs/2302.13971",
		},
		{
			Owner:       "abachaa",
			Repo:        "MedVidQA",
			Description: "MedVidQA: A dataset of medical video-based question answering",
			PaperTitle:  "MedVidQA: A Medical Video Question Answering Challenge",
			PaperURL:    "https://arxiv.org/abs/2201.12888",
		},
	}

	repos := []models.Repository{}

	// 遍历已知仓库列表
	for _, knownRepo := range knownRepos {
		repoName := fmt.Sprintf("%s/%s", knownRepo.Owner, knownRepo.Repo)

		// 创建仓库条目
		repository := models.Repository{
			Name:           repoName,
			URL:            fmt.Sprintf("https://github.com/%s", repoName),
			Description:    knownRepo.Description,
			Language:       "unknown", // 将通过enrichRepositoryWithGitHubDetails更新
			Stars:          0,         // 将通过enrichRepositoryWithGitHubDetails更新
			LastUpdated:    time.Now(),
			TechStack:      []string{"research", "ai", "paper"},
			RelevanceScore: 0.9,
			HasDocs:        true,
			Source:         "GitHub AI Papers",
			PaperURL:       knownRepo.PaperURL,
			PaperTitle:     knownRepo.PaperTitle,
		}

		// 获取GitHub仓库详细信息
		enrichRepositoryWithGitHubDetails(&repository)

		if repository.Stars > 0 {
			repos = append(repos, repository)
		}
	}

	return repos, nil
}

// enrichRepositoryWithGitHubDetails fetches additional details from GitHub
func enrichRepositoryWithGitHubDetails(repo *models.Repository) {
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
		repo.Stars = githubRepo.StargazersCount
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

	// 计算并获取模型分类
	repo.GetModelCategories()
}

// truncateString safely truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
