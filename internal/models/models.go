package models

import (
	"strings"
	"time"
)

// Repository represents a GitHub repository
type Repository struct {
	Name           string       `json:"name"`
	URL            string       `json:"url"`
	Description    string       `json:"description"`
	Language       string       `json:"language"`
	Stars          int          `json:"stars"`
	Forks          int          `json:"forks"`
	GainedStars    int          `json:"gained_stars"`
	GainedForks    int          `json:"gained_forks"`
	LastUpdated    time.Time    `json:"last_updated"`
	LastCommit     time.Time    `json:"last_commit"`
	TechStack      []string     `json:"tech_stack"`
	TrendMetrics   TrendMetrics `json:"trend_metrics"`
	RelevanceScore float64      `json:"relevance_score"`
	HasDocs        bool         `json:"has_docs"`
	HasWiki        bool         `json:"has_wiki"`   // 是否有Wiki文档
	HasReadme      bool         `json:"has_readme"` // 是否有README文档
	DocsURL        string       `json:"docs_url"`   // 文档URL
	ModelCategories []string    `json:"model_categories"` // 模型分类
	Source         string       `json:"source"`           // 数据来源，如"GitHub"、"Papers with Code"、"arXiv"
	PaperURL       string       `json:"paper_url"`        // 论文URL
	PaperTitle     string       `json:"paper_title"`      // 论文标题
	Authors        []string     `json:"authors"`          // 作者列表
}

// TrendMetrics captures trending information
type TrendMetrics struct {
	Stars24h int `json:"stars_24h"`
	Forks24h int `json:"forks_24h"`
	Views7d  int `json:"views_7d"`
}

// Paper represents a research paper
type Paper struct {
	Title                string    `json:"title"`
	URL                  string    `json:"url"`
	Authors              []string  `json:"authors"`
	PublishedDate        time.Time `json:"published_date"`
	Source               string    `json:"source"` // ArXiv, ACL, etc.
	Summary              string    `json:"summary"`
	Keywords             []string  `json:"keywords"`
	CitationCount        int       `json:"citation_count"`
	CitationVelocity     float64   `json:"citation_velocity"`
	NoveltyScore         float64   `json:"novelty_score"`         // 0-5
	ReproducibilityScore float64   `json:"reproducibility_score"` // 0-5
	CoreContributions    []string  `json:"core_contributions"`
	KeyTechniques        []string  `json:"key_techniques"`
	CodeSnippet          string    `json:"code_snippet"`
	ArchitectureDiagram  string    `json:"architecture_diagram"`
}

// DataSource represents external data source configurations
type DataSource struct {
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Type          string    `json:"type"`           // github, papers, etc.
	FetchInterval int       `json:"fetch_interval"` // in minutes
	LastFetched   time.Time `json:"last_fetched"`
	Status        string    `json:"status"` // active, error, etc.
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// Keywords for filtering GitHub repositories
var AIKeywords = []string{
	// LLM相关
	"llm", "gpt", "bert", "transformer", "nlp", "ai", "machine-learning", "ml",
	"neural-network", "deep-learning", "artificial-intelligence", "agi", "agent",
	"reinforcement-learning", "rl", "natural-language-processing", "diffusion",
	"generative-ai", "language-model", "stable-diffusion", "openai", "huggingface",
	"langchain", "chatgpt", "claude", "gemini", "mistral", "large-language-model",
	"multimodal", "rlhf", "alignment",

	// 新增更多关键词
	"ai-agent", "ai-assistant", "ai-tools", "ai-art", "ai-service",
	"prompt-engineering", "fine-tuning", "vector-database", "semantic-search",
	"embedding", "llama", "mixtral", "vicuna", "pythia", "falcon", "qwen",
	"baichuan", "glm", "ernie", "cohere", "token", "tokenizer", "attention",
	"vllm", "tei", "rag", "retrieval-augmented", "text-to-image", "text-to-video",
	"text-to-speech", "speech-to-text", "image-generation", "computer-vision",
	"vision-language", "multimodality", "knowledge-graph", "sora", "midjourney", "dall-e",
	"tensor", "neural", "gans", "gan", "vae", "diffuser", "latent", "inference",

	// 语言特定的AI关键词
	"tensorflow", "pytorch", "onnx", "jax", "keras", "mxnet",
	"scikit-learn", "pandas", "numpy", "scipy", "matplotlib",

	// C++特定的AI关键词
	"cpp-ai", "cpp-ml", "cpp-deep-learning", "libtorch", "tensorflow-cpp",
	"onnxruntime", "dlib", "opencv", "ncnn", "milvus", "faiss-cpp",
	"cuda", "cudnn", "tensorrt", "openvino", "nebula", "inference-engine",

	// GoLang特定的AI关键词
	"go-ai", "go-ml", "go-deep-learning", "gorgonia", "gonum", "gosseract",
	"onnx-go", "tfgo", "gocv", "go-milvus", "ollama", "go-llm",
	"langchaingo", "go-openai", "weaviate", "go-embedding", "go-vector",
}

// AIModelKeywords 定义主流AI模型的关键词，用于模型分类
var AIModelKeywords = map[string][]string{
	"OpenAI": {
		"chatgpt", "gpt", "gpt-3", "gpt-4", "gpt-3.5", "gpt3", "gpt4", "openai",
		"dall-e", "dall-e-2", "dall-e-3", "whisper", "claude-opus", "sora",
	},
	"Gemini": {
		"gemini", "gemini-pro", "gemini-ultra", "google-gemini", "bard",
		"google-bard", "palm", "palm-2",
	},
	"Claude": {
		"claude", "anthropic", "claude-instant", "claude-2", "claude-3",
	},
	"Mistral": {
		"mistral", "mistral-ai", "mistral-7b", "mistral-medium", "mistral-large", "mistral-small",
	},
	"Llama": {
		"llama", "llama-2", "llama-3", "llama2", "llama3", "meta-llama", "meta-ai",
	},
	"国内模型": {
		"文心一言", "文心", "百度", "ernie", "ernie-bot", "baidu", "wenxin",
		"通义千问", "通义", "阿里", "qwen", "qwen-vl", "alibaba",
		"讯飞星火", "讯飞", "xunfei", "sparkdesk",
		"腾讯混元", "hunyuan", "tencent", "混元",
		"智谱", "chatglm", "glm", "glm-4", "glm-3",
		"moonshot", "kimi", "tiangong", "天工",
	},
	"开发工具": {
		"cursor", "cody", "github-copilot", "copilot", "deepseek", "deepseek-coder",
		"tabnine", "replit", "jetbrains", "intellij", "vscode", "neovim", "vim",
	},
	"其他模型": {
		"stable-diffusion", "midjourney", "falcon", "vicuna", "pythia", "polylm",
		"yi", "phi", "phi-2", "phi-3", "cohere", "command", "command-r",
		"internlm", "internlm2", "qingyan", "aquila", "minimax",
	},
}

// FilterCriteria defines the criteria for filtering repositories
type FilterCriteria struct {
	MinStarsGrowthRate    int     // Minimum stars growth per day
	MaxDaysSinceCommit    int     // Maximum days since last commit
	RequiresDocumentation bool    // Whether complete documentation is required
	MinRelevanceScore     float64 // Minimum relevance score (0-1)
}

// GetModelCategories 检测仓库属于哪些模型分类
func (r *Repository) GetModelCategories() []string {
	if len(r.ModelCategories) > 0 {
		return r.ModelCategories
	}

	categories := make(map[string]bool)
	repoText := strings.ToLower(r.Name + " " + r.Description)

	// 检查仓库文本是否包含各个模型分类的关键词
	for category, keywords := range AIModelKeywords {
		for _, keyword := range keywords {
			if strings.Contains(repoText, strings.ToLower(keyword)) {
				categories[category] = true
				break
			}
		}
	}

	// 将分类结果转换为切片
	result := []string{}
	for category := range categories {
		result = append(result, category)
	}

	// 如果没有匹配的分类，则标记为"其他"
	if len(result) == 0 {
		result = append(result, "其他")
	}

	r.ModelCategories = result
	return result
}

// DefaultFilterCriteria returns the default filter criteria as per requirements
func DefaultFilterCriteria() FilterCriteria {
	return FilterCriteria{
		MinStarsGrowthRate:    0,     // 不要求每日增长星星数
		MaxDaysSinceCommit:    180,   // 允许更早的仓库，半年内有提交即可
		RequiresDocumentation: false, // 不要求文档
		MinRelevanceScore:     0.01,  // 进一步降低相关性要求，接近不过滤
	}
}
