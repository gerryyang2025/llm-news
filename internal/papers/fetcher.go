package papers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gerryyang2025/llm-news/internal/models"
)

// Constants for the APIs
const (
	paperswithcodeURL = "https://paperswithcode.com/api/v1/papers/?topics=language-modelling,transformer,nlp,llm,gpt,diffusion-models&page=1"
)

// FetchTopPapers fetches top AI/ML papers from multiple sources
func FetchTopPapers() ([]models.Paper, error) {
	var allPapers []models.Paper
	var errors []string

	// Fetch from Papers with Code
	pwcPapers, err := fetchPapersWithCode()
	if err != nil {
		log.Printf("Warning: Error fetching from Papers with Code: %v", err)
		errors = append(errors, fmt.Sprintf("Papers with Code: %v", err))
	} else if len(pwcPapers) > 0 {
		allPapers = append(allPapers, pwcPapers...)
	}

	// 获取其他博客和技术文章
	blogPosts, err := FetchOtherBlogPosts()
	if err != nil {
		log.Printf("Warning: Error fetching blog posts: %v", err)
		errors = append(errors, fmt.Sprintf("Blog posts: %v", err))
	} else if len(blogPosts) > 0 {
		allPapers = append(allPapers, blogPosts...)
	}

	// 如果所有数据源都获取失败，返回明确的错误，不再使用示例数据
	if len(allPapers) == 0 {
		if len(errors) > 0 {
			return nil, fmt.Errorf("failed to fetch papers from all sources: %s", strings.Join(errors, "; "))
		}
		return nil, fmt.Errorf("no papers found from any source")
	}

	// Calculate citation velocity and novelty scores
	enrichPapersWithScores(allPapers)

	// Sort papers by relevance
	sortPapersByRelevance(allPapers)

	return allPapers, nil
}

// fetchPapersWithCode fetches papers from Papers with Code
func fetchPapersWithCode() ([]models.Paper, error) {
	// Make HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second, // 增加超时时间到30秒
	}

	resp, err := client.Get(paperswithcodeURL)
	if err != nil {
		// 出错时不再返回示例数据
		return nil, fmt.Errorf("failed to fetch papers from Papers with Code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 状态码不对也不返回示例数据
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
			PublishedRaw json.RawMessage `json:"published"`
			Authors      json.RawMessage `json:"authors"`
			Abstract     string          `json:"abstract"`
			Repositories []struct {
				URL       string `json:"url"`
				Framework string `json:"framework"`
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

	papers := []models.Paper{}

	// Process each paper
	for _, result := range apiResponse.Results {
		// 解析日期，支持多种格式
		var publishedDate time.Time
		var publishedStr string

		// 尝试将原始JSON解析为字符串
		if err := json.Unmarshal(result.PublishedRaw, &publishedStr); err == nil {
			// 尝试多种日期格式
			layouts := []string{
				time.RFC3339,
				"2006-01-02",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05",
				"2006/01/02",
			}

			for _, layout := range layouts {
				if t, err := time.Parse(layout, publishedStr); err == nil {
					publishedDate = t
					break
				}
			}
		}

		// 如果无法解析日期，则使用当前时间
		if publishedDate.IsZero() {
			publishedDate = time.Now().AddDate(0, 0, -rand.Intn(30)) // 随机设定为过去30天内
			log.Printf("Warning: Could not parse date for paper %s, using estimated date", result.Title)
		}

		// 灵活处理作者字段，可能是对象数组或字符串
		authors := []string{}

		// 尝试解析为数组形式
		var authorsArray []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(result.Authors, &authorsArray); err == nil {
			for _, author := range authorsArray {
				authors = append(authors, author.Name)
			}
		} else {
			// 尝试解析为单个字符串
			var authorStr string
			if err := json.Unmarshal(result.Authors, &authorStr); err == nil {
				authors = append(authors, authorStr)
			} else {
				// 尝试解析为字符串数组
				var authorStrArray []string
				if err := json.Unmarshal(result.Authors, &authorStrArray); err == nil {
					authors = append(authors, authorStrArray...)
				} else {
					// 无法解析作者信息，使用默认作者
					authors = append(authors, "Unknown Author")
					// 记录错误，但只记录标题前20个字符，避免日志过长
					titlePreview := result.Title
					if len(titlePreview) > 20 {
						titlePreview = titlePreview[:20] + "..."
					}
					log.Printf("Warning: Could not parse authors for paper %s", titlePreview)
				}
			}
		}

		keywords := []string{}
		for _, task := range result.Tasks {
			keywords = append(keywords, task.Name)
		}

		// Create code snippet based on repositories
		codeSnippet := ""
		if len(result.Repositories) > 0 {
			repoURL := result.Repositories[0].URL
			codeSnippet = fmt.Sprintf("```python\n# Example usage from %s\nimport torch\n\n# Load model\nmodel = torch.hub.load('%s', 'default')\noutputs = model(inputs)\n```", repoURL, strings.TrimPrefix(repoURL, "https://github.com/"))
		}

		paper := models.Paper{
			Title:         result.Title,
			URL:           result.URL,
			Authors:       authors,
			PublishedDate: publishedDate,
			Source:        "Papers with Code",
			Summary:       result.Abstract,
			Keywords:      keywords,
			CitationCount: rand.Intn(50) + 10, // Simulate citation count (we don't have real data)
			CodeSnippet:   codeSnippet,
		}

		papers = append(papers, paper)
	}

	return papers, nil
}

// enhancePaperWithDetails adds more detailed information to a paper
func enhancePaperWithDetails(paper *models.Paper) {
	// In a production environment, this would call external APIs like Semantic Scholar
	// or use NLP techniques to extract more detailed information

	// For now, we'll use a simple heuristic based on title and summary length
	titleLength := len(strings.Split(strings.ToLower(paper.Title), " "))
	summaryLength := 0
	if paper.Summary != "" {
		summaryLength = len(strings.Split(strings.ToLower(paper.Summary), " "))
	}

	// 使用标题长度和摘要长度来稍微调整一下引用数量，使其更具多样性
	citationAdjust := (titleLength % 5) + (summaryLength % 10)

	// For citation count - in the future this should come from a real API
	// We'll keep a modest count between 1-50 based on publication recency
	daysOld := time.Since(paper.PublishedDate).Hours() / 24
	if daysOld < 30 {
		paper.CitationCount = rand.Intn(20) + 1 + citationAdjust // Newer papers have fewer citations
	} else {
		paper.CitationCount = rand.Intn(30) + 20 + citationAdjust // Older papers have more citations
	}

	// Calculate citation velocity (citations per day since publication)
	daysSincePublication := int(time.Since(paper.PublishedDate).Hours() / 24)
	if daysSincePublication < 1 {
		daysSincePublication = 1
	}
	paper.CitationVelocity = float64(paper.CitationCount) / float64(daysSincePublication)

	// Calculate novelty score (0-5) based on keywords and title analysis
	noveltyTerms := []string{"new", "novel", "first", "innovative", "breakthrough", "state-of-the-art",
		"sota", "cutting-edge", "pioneering", "groundbreaking", "unprecedented"}

	noveltyScore := 3.0 // Base score
	for _, term := range noveltyTerms {
		if containsAny(paper.Title, term) {
			noveltyScore += 0.3
		}
		if paper.Summary != "" && containsAny(paper.Summary, term) {
			noveltyScore += 0.2
		}
	}
	// Cap the score at 5.0
	if noveltyScore > 5.0 {
		noveltyScore = 5.0
	}
	paper.NoveltyScore = noveltyScore

	// Calculate reproducibility score (0-5) based on content analysis
	reproducibilityTerms := []string{"code", "github", "implementation", "dataset", "public",
		"available", "open-source", "repository", "replicate", "reproduce"}

	reproducibilityScore := 2.5 // Base score
	for _, term := range reproducibilityTerms {
		if containsAny(paper.Title, term) {
			reproducibilityScore += 0.3
		}
		if paper.Summary != "" && containsAny(paper.Summary, term) {
			reproducibilityScore += 0.2
		}
	}
	// Cap the score at 5.0
	if reproducibilityScore > 5.0 {
		reproducibilityScore = 5.0
	}
	paper.ReproducibilityScore = reproducibilityScore

	// Extract core contributions from summary
	sentences := strings.Split(paper.Summary, ". ")
	contributions := []string{}
	for i, sentence := range sentences {
		if i <= 2 && len(sentence) > 10 {
			contributions = append(contributions, sentence+".")
		}
	}
	paper.CoreContributions = contributions

	// Generate key techniques (based on keywords and title)
	techniques := []string{}
	aiTerms := []string{"transformer", "attention mechanism", "fine-tuning", "reinforcement learning",
		"diffusion model", "generative model", "multi-modal", "RLHF", "contrastive learning"}

	// Check title and keywords for AI terms
	for _, term := range aiTerms {
		if containsAny(paper.Title, term) || containsAnyInSlice(paper.Keywords, term) {
			techniques = append(techniques, term)
		}
		if len(techniques) >= 3 {
			break
		}
	}

	// If we don't have enough techniques, add some based on the category
	if len(techniques) < 2 {
		for _, keyword := range paper.Keywords {
			if keyword == "cs.CL" {
				techniques = append(techniques, "Natural Language Processing")
			} else if keyword == "cs.CV" {
				techniques = append(techniques, "Computer Vision")
			} else if keyword == "cs.AI" {
				techniques = append(techniques, "Artificial Intelligence")
			} else if keyword == "cs.LG" {
				techniques = append(techniques, "Machine Learning")
			}
		}
	}

	paper.KeyTechniques = techniques

	// 检测主流模型关键词并添加到keywords中
	modelKeywords := map[string][]string{
		"Cursor":   {"cursor", "cursor ai"},
		"DeepSeek": {"deepseek", "deepseek-coder", "deepseek-llm"},
		"Hunyuan":  {"hunyuan", "tencent hunyuan"},
		"文心一言":     {"文心一言", "wenxin", "ernie bot", "ernie", "baidu"},
		"GPT":      {"gpt", "gpt-4", "gpt-3", "chatgpt", "openai"},
		"Claude":   {"claude", "anthropic"},
		"Gemini":   {"gemini", "google gemini"},
		"Llama":    {"llama", "meta llama", "llama 2", "llama 3"},
		"Mistral":  {"mistral ai", "mistral"},
		"Qwen":     {"qwen", "tongyi qianwen", "通义千问"},
		"ChatGLM":  {"chatglm", "glm", "智谱"},
	}

	// 当前关键词转为小写，便于匹配
	lowerKeywords := make([]string, len(paper.Keywords))
	for i, k := range paper.Keywords {
		lowerKeywords[i] = strings.ToLower(k)
	}

	// 检查标题、摘要和现有关键词中是否包含模型关键词
	lowerTitle := strings.ToLower(paper.Title)
	lowerSummary := strings.ToLower(paper.Summary)

	for model, terms := range modelKeywords {
		for _, term := range terms {
			// 如果标题、摘要或现有关键词中包含模型关键词
			if strings.Contains(lowerTitle, term) ||
				strings.Contains(lowerSummary, term) ||
				containsTermInSlice(lowerKeywords, term) {
				// 检查模型是否已经在关键词中
				modelInKeywords := false
				for _, k := range paper.Keywords {
					if strings.EqualFold(k, model) {
						modelInKeywords = true
						break
					}
				}

				// 如果模型还不在关键词中，添加它
				if !modelInKeywords {
					paper.Keywords = append(paper.Keywords, model)
				}

				// 已添加此模型，无需检查此模型的其他关键词
				break
			}
		}
	}

	// Generate example code snippet if relevant
	if containsAny(paper.Title, "code", "implementation", "github") ||
		containsAny(paper.Summary, "code", "implementation", "github") {
		paper.CodeSnippet = generateCodeSnippet(paper)
	}
}

// 检查字符串数组中是否包含指定术语
func containsTermInSlice(slice []string, term string) bool {
	for _, item := range slice {
		if strings.Contains(item, term) {
			return true
		}
	}
	return false
}

// containsAny checks if a string contains any of the specified terms
func containsAny(text string, terms ...string) bool {
	lowerText := strings.ToLower(text)
	for _, term := range terms {
		if strings.Contains(lowerText, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// containsAnySlice checks if any string in a slice contains any of the specified terms
func containsAnySlice(texts []string, terms ...string) bool {
	for _, text := range texts {
		if containsAny(text, terms...) {
			return true
		}
	}
	return false
}

// containsAnyInSlice 检查一个字符串数组中的任何元素是否包含指定的术语
func containsAnyInSlice(texts []string, term string) bool {
	for _, text := range texts {
		if strings.Contains(strings.ToLower(text), strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// generateCodeSnippet creates a sample code snippet based on the paper topic
func generateCodeSnippet(paper *models.Paper) string {
	if containsAny(paper.Title, "LLM", "GPT", "language model") {
		return "```python\n# Example LLM implementation\nimport transformers\n\nmodel = transformers.AutoModelForCausalLM.from_pretrained(\"gpt2\")\ntokenizer = transformers.AutoTokenizer.from_pretrained(\"gpt2\")\n\ninputs = tokenizer(\"Hello, I'm a language model\", return_tensors=\"pt\")\noutputs = model.generate(**inputs, max_length=50)\nprint(tokenizer.decode(outputs[0]))\n```"
	} else if containsAny(paper.Title, "vision", "image", "CV") {
		return "```python\n# Example Vision Transformer\nimport torch\nfrom transformers import ViTForImageClassification, ViTImageProcessor\n\nmodel = ViTForImageClassification.from_pretrained(\"google/vit-base-patch16-224\")\nprocessor = ViTImageProcessor.from_pretrained(\"google/vit-base-patch16-224\")\n\nimage = Image.open(\"image.jpg\")\ninputs = processor(images=image, return_tensors=\"pt\")\nwith torch.no_grad():\n    outputs = model(**inputs)\n```"
	} else if containsAny(paper.Title, "diffusion", "stable diffusion") {
		return "```python\n# Example Diffusion Model\nfrom diffusers import StableDiffusionPipeline\nimport torch\n\npipe = StableDiffusionPipeline.from_pretrained(\"runwayml/stable-diffusion-v1-5\")\npipe = pipe.to(\"cuda\")\n\nprompt = \"a photo of an astronaut riding a horse on mars\"\nimage = pipe(prompt).images[0]\nimage.save(\"astronaut_rides_horse.png\")\n```"
	} else {
		return "```python\n# Example implementation\nimport torch\nimport numpy as np\n\nclass Model(torch.nn.Module):\n    def __init__(self):\n        super().__init__()\n        self.linear = torch.nn.Linear(10, 1)\n        \n    def forward(self, x):\n        return self.linear(x)\n\nmodel = Model()\ninputs = torch.randn(1, 10)\noutputs = model(inputs)\n```"
	}
}

// enrichPapersWithScores calculates additional scores for all papers
func enrichPapersWithScores(papers []models.Paper) {
	// Seed random for consistent results in demo
	rand.Seed(time.Now().UnixNano())

	for i := range papers {
		// If we haven't already set these values
		if papers[i].CitationCount == 0 {
			papers[i].CitationCount = rand.Intn(100) + 1
		}

		if papers[i].CitationVelocity == 0 {
			daysSincePublication := int(time.Since(papers[i].PublishedDate).Hours() / 24)
			if daysSincePublication < 1 {
				daysSincePublication = 1
			}
			papers[i].CitationVelocity = float64(papers[i].CitationCount) / float64(daysSincePublication)
		}

		if papers[i].NoveltyScore == 0 {
			noveltyScore := 3.0 + (rand.Float64() * 2.0) // Between 3.0 and 5.0
			papers[i].NoveltyScore = noveltyScore
		}

		if papers[i].ReproducibilityScore == 0 {
			reproducibilityScore := 2.0 + (rand.Float64() * 3.0) // Between 2.0 and 5.0
			papers[i].ReproducibilityScore = reproducibilityScore
		}

		// Add other scores and details as needed
		if len(papers[i].CoreContributions) == 0 {
			enhancePaperWithDetails(&papers[i])
		}
	}
}

// sortPapersByRelevance sorts papers by a combination of factors for maximum relevance
func sortPapersByRelevance(papers []models.Paper) {
	// 使用sort包进行高效排序
	sort.Slice(papers, func(i, j int) bool {
		// 计算综合评分（考虑多个因素的加权平均）
		// 1. 引用速度 (30%)
		// 2. 新颖性分数 (30%)
		// 3. 引用总数 (25%)
		// 4. 发布日期新鲜度 (15%)

		// 计算日期新鲜度分数（越近越高，最高5分）
		daysOldI := time.Since(papers[i].PublishedDate).Hours() / 24
		daysOldJ := time.Since(papers[j].PublishedDate).Hours() / 24

		freshnessI := 5.0 - math.Min(daysOldI/60, 5.0) // 60天内线性递减，最低0分
		freshnessJ := 5.0 - math.Min(daysOldJ/60, 5.0)

		// 综合评分计算
		scoreI := (papers[i].CitationVelocity * 0.3) +
			(papers[i].NoveltyScore * 0.3) +
			(float64(papers[i].CitationCount) / 100.0 * 0.25) +
			(freshnessI * 0.15)

		scoreJ := (papers[j].CitationVelocity * 0.3) +
			(papers[j].NoveltyScore * 0.3) +
			(float64(papers[j].CitationCount) / 100.0 * 0.25) +
			(freshnessJ * 0.15)

		// 降序排列（高分在前）
		return scoreI > scoreJ
	})
}
