
# 🗺️ BizAtlas

<p align="center">
  <strong>Google Maps Business Intelligence & Lead Discovery</strong>
</p>

<p align="center">
  Discover businesses. Extract insights. Transform location data into opportunities.
</p>

<p align="center">
  <a href="https://github.com/ATLURI0001/BizAtlas">
    <img src="https://img.shields.io/badge/Project-BizAtlas-blue?style=flat-square" alt="BizAtlas">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="MIT License">
  </a>
  <img src="https://img.shields.io/badge/Language-Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Interface-CLI%20%7C%20Web%20%7C%20API-6366F1?style=flat-square" alt="Interfaces">
</p>

---

## 📖 Overview

**BizAtlas** is an open-source, Go-based business data extraction tool designed to collect and organize publicly accessible business information from Google Maps.

It helps developers, researchers, and businesses discover local establishments, collect contact details, analyze business listings, and export structured datasets.

BizAtlas provides multiple interfaces, configurable scraping options, concurrent processing, and flexible export capabilities.

### Use Cases

- **Lead Generation** — Discover potential business customers.
- **Market Research** — Analyze businesses across locations and industries.
- **Business Intelligence** — Collect structured information for analysis.
- **Data Enrichment** — Enhance existing datasets with business details.
- **Local SEO Research** — Examine listings, ratings, and reviews.
- **Automation** — Integrate business discovery into applications.

---

## ✨ Features

| Feature | Description |
|---|---|
| Business Discovery | Search Google Maps by keywords and locations |
| Contact Extraction | Collect addresses, websites, and phone numbers |
| Email Discovery | Optionally extract emails from business websites |
| Review Collection | Retrieve ratings, review counts, and reviews |
| Geolocation | Extract coordinates, place IDs, and location information |
| Multiple Exports | CSV, JSON, and supported integrations |
| Concurrent Scraping | Process multiple searches simultaneously |
| Proxy Support | HTTP, HTTPS, and SOCKS proxies |
| Resume Support | Continue supported interrupted scraping jobs |
| Fast Mode | Reduced-data extraction for quicker searches |
| Web Interface | Manage scraping jobs through a browser |
| REST API | Integrate scraping workflows into applications |
| Database Integration | PostgreSQL-backed workflows |
| Custom Writers | Extend output handling using plugins |

---

## 🧭 Navigation

- [Quick Start](#-quick-start)
- [Web Interface](#-web-interface)
- [REST API](#-rest-api)
- [Extracted Data](#-extracted-data)
- [Configuration](#-configuration)
- [Advanced Usage](#-advanced-usage)
- [Performance](#-performance)
- [Project Structure](#-project-structure)
- [Contributing](#-contributing)
- [Maintainer](#-maintainer)
- [License](#-license)

---

## 🚀 Quick Start

BizAtlas supports command-line execution, a local web interface, and REST API access.

### Clone the Repository

```bash
git clone https://github.com/ATLURI0001/BizAtlas.git
cd BizAtlas
```

### Prepare Search Queries

Create a file named `queries.txt`.

```text
restaurants in Hyderabad
software companies in Bengaluru
hotels in Mumbai
coffee shops in Pune
```

Each line represents a separate search query.

### Command-Line Execution

Run the compiled application with your input file.

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -depth 1 \
  -exit-on-inactivity 3m
```

On Windows, use the corresponding `.exe` executable.

> **Note:** The existing executable may still be named `google-maps-scraper`. Renaming the GitHub repository does not automatically rename compiled binaries.

### Export JSON

```bash
./google-maps-scraper \
  -input queries.txt \
  -json \
  -results results.json \
  -depth 1
```

### Extract Email Addresses

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -email
```

Email extraction visits business websites and may increase processing time.

---

## 🌐 Web Interface

BizAtlas includes a browser-based interface for managing scraping jobs.

Start the web server:

```bash
./google-maps-scraper -web -addr :8080
```

Open:

**http://localhost:8080**

The interface allows users to create scraping jobs, monitor their progress, and access collected results.

---

## 🔌 REST API

BizAtlas provides a REST API for programmatic scraping management.

### Available Endpoints

| Method | Endpoint | Description |
|---|---|---|
| POST | `/api/v1/jobs` | Create a scraping job |
| GET | `/api/v1/jobs` | List scraping jobs |
| GET | `/api/v1/jobs/{id}` | Retrieve job details |
| DELETE | `/api/v1/jobs/{id}` | Delete a job |
| GET | `/api/v1/jobs/{id}/download` | Download results |

### API Documentation

When the API documentation is enabled, access it at:

```text
http://localhost:8080/api/docs
```

The API can be used with custom applications, automation scripts, and data-processing pipelines.

---

## 📊 Extracted Data

BizAtlas supports extraction of up to 36 business-related fields, depending on listing availability and enabled options.

<details>
<summary><strong>View all 36 supported fields</strong></summary>

| # | Field | Description |
|---|---|---|
| 1 | `input_id` | Input query identifier |
| 2 | `link` | Google Maps listing URL |
| 3 | `title` | Business name |
| 4 | `category` | Business category |
| 5 | `address` | Business address |
| 6 | `open_hours` | Operating hours |
| 7 | `popular_times` | Visitor activity information |
| 8 | `website` | Business website |
| 9 | `phone` | Contact phone number |
| 10 | `plus_code` | Google Maps Plus Code |
| 11 | `review_count` | Number of reviews |
| 12 | `review_rating` | Average rating |
| 13 | `reviews_per_rating` | Rating distribution |
| 14 | `latitude` | Geographic latitude |
| 15 | `longitude` | Geographic longitude |
| 16 | `cid` | Google Maps CID |
| 17 | `status` | Business operating status |
| 18 | `descriptions` | Business description |
| 19 | `reviews_link` | Reviews URL |
| 20 | `thumbnail` | Thumbnail URL |
| 21 | `timezone` | Business timezone |
| 22 | `price_range` | Price category |
| 23 | `data_id` | Google Maps data identifier |
| 24 | `street_view_url` | Street View URL |
| 25 | `place_id` | Google Maps Place ID |
| 26 | `images` | Business image URLs |
| 27 | `reservations` | Reservation URL |
| 28 | `order_online` | Online ordering URL |
| 29 | `menu` | Menu URL |
| 30 | `owner` | Ownership information |
| 31 | `complete_address` | Structured address |
| 32 | `credit_cards_accepted` | Accepted payment networks |
| 33 | `about` | Additional business information |
| 34 | `user_reviews` | Customer reviews |
| 35 | `user_reviews_extended` | Extended reviews |
| 36 | `emails` | Extracted email addresses |

</details>

Not every listing contains all fields. Email and extended review extraction require their corresponding options.

---

## ⚙️ Configuration

BizAtlas provides configurable options for search behavior, output formats, concurrency, and networking.

### Core Options

| Flag | Description |
|---|---|
| `-input` | Input queries file |
| `-results` | Output file path |
| `-json` | Export JSON |
| `-resume` | Resume interrupted scraping |
| `-depth` | Maximum scrolling depth |
| `-c` | Concurrent scraping jobs |
| `-lang` | Search language |
| `-geo` | Geographic coordinates |
| `-zoom` | Map zoom level |
| `-radius` | Search radius |
| `-email` | Enable email extraction |
| `-extra-reviews` | Collect extended reviews |
| `-fast-mode` | Enable reduced-data fast mode |

### Networking and Performance

| Flag | Description |
|---|---|
| `-proxies` | Proxy URLs |
| `-proxies-file` | Proxy configuration file |
| `-browser-pool-size` | Browser process count |
| `-pages-per-browser` | Concurrent browser tabs |
| `-exit-on-inactivity` | Stop after inactivity |
| `-debug` | Browser debugging mode |

### Web and Database

| Flag | Description |
|---|---|
| `-web` | Enable web server |
| `-addr` | Server address |
| `-data-folder` | Web application data directory |
| `-dsn` | PostgreSQL connection string |
| `-produce` | Generate database-backed jobs |
| `-writer` | Custom writer plugin |

For the complete options supported by your executable:

```bash
./google-maps-scraper -h
```

---

## 🛠️ Advanced Usage

### Resume Interrupted Scraping

Continue a supported interrupted job:

```bash
./google-maps-scraper \
  -resume \
  -input queries.txt \
  -results results.csv \
  -depth 10
```

Use the same input and scraping configuration when resuming.

### Configure Concurrency

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -c 8 \
  -browser-pool-size 2 \
  -pages-per-browser 4
```

Concurrency improves parallel processing but increases CPU and memory usage.

### Proxy Configuration

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -proxies "http://host:port,socks5://host2:port2"
```

Supported proxy protocols include HTTP, HTTPS, SOCKS5, and SOCKS5H.

### Geographic Search

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -geo "17.3850,78.4867" \
  -radius 5000 \
  -zoom 15 \
  -fast-mode
```

This example targets an area around Hyderabad.

### Grid-Based Scraping

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -grid-bbox "17.35,78.43,17.43,78.53" \
  -grid-cell 0.5 \
  -zoom 16 \
  -depth 1
```

Grid scraping divides a geographic bounding box into smaller search areas.

### PostgreSQL Integration

BizAtlas supports PostgreSQL-backed workflows for distributed processing.

```bash
./google-maps-scraper \
  -dsn "postgres://user:password@localhost:5432/bizatlas" \
  -produce \
  -input queries.txt
```

Database provisioning and configuration must be completed before using this feature.

---

## 📈 Performance

Scraping performance depends on hardware, network conditions, concurrency, search depth, and the availability of Google Maps data.

### Optimization Options

| Setting | Purpose |
|---|---|
| Concurrency | Process multiple search jobs |
| Browser Pool | Control browser process usage |
| Pages per Browser | Improve resource efficiency |
| Search Depth | Control result collection |
| Fast Mode | Reduce extraction overhead |
| Proxy Configuration | Configure network routing |

For larger workloads, monitor CPU, memory, browser utilization, and request failures.

---

## 📁 Project Structure

BizAtlas includes components for scraping, data processing, web interfaces, API services, and deployment integrations.

The repository contains:

- Go source code
- Command-line scraping functionality
- Web application components
- REST API functionality
- Configuration files
- Documentation and examples
- Database integrations
- Container and deployment configurations

Refer to the repository tree for the current file structure.

---

## 🤝 Contributing

Contributions, suggestions, and bug reports are welcome.

To contribute:

1. Open an issue describing your proposal.
2. Create a development branch.
3. Implement and test your changes.
4. Submit a pull request.
5. Provide relevant documentation where necessary.

Please keep contributions focused, maintainable, and consistent with the project's existing architecture.

---

## 👤 Maintainer

**Ramprasad Atluri**

GitHub: [@ATLURI0001](https://github.com/ATLURI0001)

Repository: [BizAtlas](https://github.com/ATLURI0001/BizAtlas)

---

## 📄 License

BizAtlas is distributed under the [MIT License](LICENSE).

The project incorporates existing open-source code. Applicable original copyright and license notices must be retained.

---

## ⚖️ Responsible Use

BizAtlas is intended for legitimate research, business intelligence, and automation.

Users are responsible for complying with applicable laws, privacy regulations, website terms, and data-use restrictions.

Publicly accessible information is not necessarily unrestricted for automated collection or redistribution.

---

<p align="center">
  <strong>BizAtlas</strong>
</p>

<p align="center">
  <sub>Business Discovery • Location Intelligence • Data Extraction</sub>
</p>

<p align="center">
  Maintained by <a href="https://github.com/ATLURI0001">Ramprasad Atluri</a>
</p>
