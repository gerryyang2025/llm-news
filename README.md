# LLM News

LLM News is an automated information collection system that tracks the latest developments in AI/ML, focusing on Large Language Models (LLMs), AGI, and related technologies. The system regularly scrapes GitHub trending repositories and collects top research papers from ArXiv to keep you updated with the most recent advancements.

## Features

- **GitHub Trending Scraper**: Automatically scrapes GitHub trending repositories every 6 hours, filtering for AI/ML-related projects using keywords like LLM, AGI, Agent, etc.
- **Research Paper Collector**: Fetches the latest AI research papers from ArXiv daily, focusing on the most relevant and impactful papers.
- **Clean Web Interface**: Presents the collected information in a user-friendly web interface.
- **Advanced Filtering Options**:
  - Filter repositories by categories (LLMs, Agents, Multimodal, Diffusion)
  - Filter by specific AI models (Gemini, Claude, Llama, GPT, etc.) to find model-specific repositories
  - Official repositories are highlighted with badges for easy identification
- **API Endpoints**: Provides JSON API endpoints for accessing the data programmatically.
- **Network Accessibility**: Automatically binds to your machine's IP address, allowing other devices on the same network to access the service.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/gerryyang2025/llm-news.git
   cd llm-news
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

### Building the Application

We provide scripts to build the application into a standalone binary:

1. **Build for current platform** (creates bin/llm-news):
   ```bash
   ./scripts/build.sh
   ```

2. **Cross-platform build** (creates binaries for multiple platforms):
   ```bash
   ./scripts/cross-build.sh
   ```

The build scripts will create binaries in the `bin` directory:
- `bin/llm-news` - Binary for the current platform
- `bin/llm-news-<os>-<arch>` - Platform-specific binaries (when using cross-build)

### Running the Service

You have four options to run the LLM News service:

#### Option 1: Direct Execution

Run the application directly (for development or testing):
```bash
go run cmd/server/main.go
```

Then open your browser and navigate to `http://<your-ip>:8081` where `<your-ip>` is your machine's IP address. The application will display this address when it starts.

#### Option 2: Using Management Scripts

We provide convenient scripts to manage the service:

1. **Start the service** (runs in background and logs to file):
   ```bash
   ./scripts/start.sh
   ```

   *Note: The start script will automatically use the compiled binary if available, otherwise it will use `go run`. It will also display the correct IP address to access the service.*

2. **Check service status**:
   ```bash
   ./scripts/status.sh
   ```

3. **Stop the service**:
   ```bash
   ./scripts/stop.sh
   ```

#### Option 3: Run the Compiled Binary

If you've built the application:
```bash
./bin/llm-news
```

The service will be accessible at `http://<your-ip>:8081` when running, where `<your-ip>` is your machine's IP address. The application will display this address in the console when it starts.

#### Option 4: Deploy as a System Service (Linux with systemd)

For production environments, you can set up LLM News as a systemd service:

1. Build the application first:
   ```bash
   ./scripts/build.sh
   ```

2. Edit the service configuration file:
   ```bash
   cp scripts/llm-news.service /tmp/llm-news.service
   nano /tmp/llm-news.service
   ```

   Update the `User`, `WorkingDirectory`, and `ExecStart` fields with appropriate values. For the `ExecStart`, use the absolute path to the binary:
   ```
   ExecStart=/absolute/path/to/llm-news/bin/llm-news
   ```

3. Install the service:
   ```bash
   sudo mv /tmp/llm-news.service /etc/systemd/system/
   sudo systemctl daemon-reload
   ```

4. Enable and start the service:
   ```bash
   sudo systemctl enable llm-news
   sudo systemctl start llm-news
   ```

5. Check service status:
   ```bash
   sudo systemctl status llm-news
   ```

The service will automatically start on system boot and will be accessible at `http://<your-ip>:8081`.

#### Option 5: Deploy with Docker

If you prefer using Docker, we provide a Dockerfile and docker-compose.yml for easy deployment:

1. Build and start the service using Docker Compose:
   ```bash
   docker-compose up -d
   ```

2. Check the container status:
   ```bash
   docker-compose ps
   ```

3. View logs:
   ```bash
   docker-compose logs -f
   ```

4. Stop the service:
   ```bash
   docker-compose down
   ```

The service will be accessible at `http://<your-ip>:8081` when running. If you want to access it from outside the Docker network, make sure the ports are properly mapped in your docker-compose.yml file.

## Network Access

By default, the application now binds to your machine's actual IP address instead of localhost (127.0.0.1). This means:

1. **Local Network Access**: Other devices on the same network can access the service using your machine's IP address.
2. **IP Discovery**: The application automatically detects and displays your IP address when starting.
3. **Multiple Network Interfaces**: If your machine has multiple network interfaces, the application will try to select the most appropriate one.
4. **Firewall Considerations**: Make sure your firewall allows incoming connections on port 8081 if you want other devices to access the service.

If you want to change the default port (8081), you'll need to modify the port number in the `cmd/server/main.go` file and rebuild the application.

## Using the Web Interface

### Repository Filtering

The web interface provides several filtering options to help you find relevant repositories:

1. **Main Category Filters**:
   - Use the top button row to filter by main categories (All, LLMs, Agents, Multimodal, Diffusion)
   - These filters look for keywords in repository names and descriptions

2. **Model-Specific Filtering**:
   - Click on the "Models ▼" dropdown to see a list of specific AI models
   - Select a model (like Gemini, Claude, Llama) to see repositories related to that model
   - Official repositories for each model are automatically tagged with an "Official" badge

3. **Filter Results Count**:
   - The interface displays how many repositories match your current filter

4. **Filter Combinations**:
   - You can first apply a main category filter, then refine with a model filter
   - To reset filters, click the "All" button

### Paper Filtering

The research papers section also offers filtering options:
- Filter papers by topic (LLMs, Agents, Multimodal, Diffusion)
- Sort papers by Novelty, Date, or Citations

## Project Structure

```
llm-news/
├── bin/                    # Compiled binaries
├── cmd/
│   └── server/
│       └── main.go         # Main application entry point
├── internal/
│   ├── models/
│   │   └── models.go       # Data models
│   ├── papers/
│   │   └── fetcher.go      # Research paper fetching logic
│   └── scrapers/
│       └── github.go       # GitHub trending scraper
├── scripts/
│   ├── build.sh            # Script to build the application
│   ├── cross-build.sh      # Script to build for multiple platforms
│   ├── llm-news.service    # Systemd service configuration
│   ├── start.sh            # Script to start the service
│   ├── status.sh           # Script to check service status
│   └── stop.sh             # Script to stop the service
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css   # Styling
│   │   └── js/
│   │       └── main.js     # Client-side JavaScript including filtering logic
│   └── templates/
│       └── index.html      # HTML template
├── Dockerfile              # Docker container definition
├── docker-compose.yml      # Docker Compose configuration
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies checksum
└── README.md               # Project documentation
```

## API Endpoints

- `GET /api/repos` - Returns JSON array of trending GitHub repositories
- `GET /api/papers` - Returns JSON array of research papers

## Customization

### Adding More Keywords

To add more keywords for filtering GitHub repositories, edit the `AIKeywords` slice in `internal/models/models.go`.

### Adding Official Repositories

To add more official repositories for model-specific filtering, edit the `officialRepos` object in `web/static/js/main.js`:

```javascript
const officialRepos = {
    'cursor': ['getcursor'],
    'deepseek': ['deepseek-ai'],
    'claude': ['anthropic'],
    'gemini': ['google-gemini'],
    'llama': ['meta-llama'],
    'qwen': ['QwenLM'],
    'hunyuan': ['HunyuanVideo', 'HunyuanDiT'],
};
```

### Changing Scraping Frequency

To change how often the system scrapes for new data, modify the scheduler settings in `cmd/server/main.go`.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [GoQuery](https://github.com/PuerkitoBio/goquery)
- [Gocron](https://github.com/go-co-op/gocron)
