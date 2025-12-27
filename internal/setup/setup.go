package setup

import (
	"database/sql"
	"fmt"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type SetupEndpoints struct {
	db          *sql.DB
	externalURL string
}

func NewSetupEndpoints(db *sql.DB, externalURL string) *SetupEndpoints {
	return &SetupEndpoints{
		db:          db,
		externalURL: externalURL,
	}
}

// HasOwner checks if an owner has been registered
func (s *SetupEndpoints) HasOwner() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'owner'").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check owner: %w", err)
	}
	return count > 0, nil
}

// LandingPage serves the setup landing page when no owner is registered
// After owner registration, returns 404
func (s *SetupEndpoints) LandingPage(ctx *fasthttp.RequestCtx) {
	hasOwner, err := s.HasOwner()
	if err != nil {
		log.Error().Err(err).Msg("Failed to check owner status")
		ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}

	if hasOwner {
		// Owner exists - no landing page
		ctx.Error("Not Found", fasthttp.StatusNotFound)
		return
	}

	// Serve the setup landing page
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Prappser Server Ready</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%%, #16213e 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
            padding: 20px;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 40px;
            max-width: 500px;
            width: 100%%;
            text-align: center;
            border: 1px solid rgba(255, 255, 255, 0.2);
        }
        .icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            font-size: 28px;
            margin-bottom: 10px;
            font-weight: 600;
        }
        .subtitle {
            color: rgba(255, 255, 255, 0.7);
            margin-bottom: 30px;
            font-size: 16px;
        }
        .url-box {
            background: rgba(0, 0, 0, 0.3);
            border-radius: 12px;
            padding: 16px;
            margin-bottom: 20px;
        }
        .url-label {
            font-size: 12px;
            color: rgba(255, 255, 255, 0.6);
            margin-bottom: 8px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .url-container {
            display: flex;
            gap: 10px;
            align-items: center;
        }
        .url-input {
            flex: 1;
            background: rgba(255, 255, 255, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.2);
            border-radius: 8px;
            padding: 12px 16px;
            color: #fff;
            font-family: monospace;
            font-size: 14px;
            width: 100%%;
        }
        .copy-btn {
            background: #4CAF50;
            color: white;
            border: none;
            border-radius: 8px;
            padding: 12px 20px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            transition: background 0.2s;
            white-space: nowrap;
        }
        .copy-btn:hover {
            background: #45a049;
        }
        .copy-btn.copied {
            background: #2196F3;
        }
        .instructions {
            background: rgba(76, 175, 80, 0.2);
            border: 1px solid rgba(76, 175, 80, 0.3);
            border-radius: 12px;
            padding: 20px;
            margin-top: 20px;
            text-align: left;
        }
        .instructions h3 {
            font-size: 14px;
            margin-bottom: 12px;
            color: #4CAF50;
        }
        .step {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            margin-bottom: 10px;
            font-size: 14px;
            color: rgba(255, 255, 255, 0.9);
        }
        .step-num {
            background: rgba(76, 175, 80, 0.3);
            width: 24px;
            height: 24px;
            border-radius: 50%%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            font-weight: bold;
            flex-shrink: 0;
        }
        .prappser-link {
            display: inline-block;
            margin-top: 20px;
            color: #64B5F6;
            text-decoration: none;
            font-weight: 500;
        }
        .prappser-link:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">&#127881;</div>
        <h1>Your Server is Ready!</h1>
        <p class="subtitle">Copy the URL below and paste it in the Prappser app to continue setup.</p>

        <div class="url-box">
            <div class="url-label">Server URL</div>
            <div class="url-container">
                <input type="text" class="url-input" id="serverUrl" value="%s" readonly>
                <button class="copy-btn" id="copyBtn" onclick="copyUrl()">Copy</button>
            </div>
        </div>

        <div class="instructions">
            <h3>Next Steps</h3>
            <div class="step">
                <span class="step-num">1</span>
                <span>Copy the server URL above</span>
            </div>
            <div class="step">
                <span class="step-num">2</span>
                <span>Go back to the Prappser app</span>
            </div>
            <div class="step">
                <span class="step-num">3</span>
                <span>Paste the URL to connect</span>
            </div>
        </div>

        <a href="https://prappser.app" target="_blank" class="prappser-link">Open Prappser App &rarr;</a>
    </div>

    <script>
        function copyUrl() {
            const urlInput = document.getElementById('serverUrl');
            const copyBtn = document.getElementById('copyBtn');

            navigator.clipboard.writeText(urlInput.value).then(function() {
                copyBtn.textContent = 'Copied!';
                copyBtn.classList.add('copied');

                setTimeout(function() {
                    copyBtn.textContent = 'Copy';
                    copyBtn.classList.remove('copied');
                }, 2000);
            }).catch(function(err) {
                // Fallback for older browsers
                urlInput.select();
                document.execCommand('copy');
                copyBtn.textContent = 'Copied!';
                copyBtn.classList.add('copied');

                setTimeout(function() {
                    copyBtn.textContent = 'Copy';
                    copyBtn.classList.remove('copied');
                }, 2000);
            });
        }
    </script>
</body>
</html>`, s.externalURL)

	ctx.SetBody([]byte(html))
}

// SetRailwayToken stores the Railway API token for server self-management
// This endpoint requires owner authentication
func (s *SetupEndpoints) SetRailwayToken(ctx *fasthttp.RequestCtx) {
	// Parse request body
	var req struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to parse railway token request")
		ctx.Error("Invalid request body", fasthttp.StatusBadRequest)
		return
	}

	if req.Token == "" {
		ctx.Error("Token is required", fasthttp.StatusBadRequest)
		return
	}

	// Store the token in the setup_config table
	_, err := s.db.Exec(`
		INSERT INTO setup_config (id, railway_token) VALUES ('default', ?)
		ON CONFLICT(id) DO UPDATE SET railway_token = excluded.railway_token
	`, req.Token)

	if err != nil {
		log.Error().Err(err).Msg("Failed to store railway token")
		ctx.Error("Failed to store token", fasthttp.StatusInternalServerError)
		return
	}

	log.Info().Msg("Railway token stored successfully")

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(map[string]string{
		"message": "Railway token stored successfully",
	})
}

// GetRailwayToken retrieves the stored Railway token (for internal use)
func (s *SetupEndpoints) GetRailwayToken() (string, error) {
	var token sql.NullString
	err := s.db.QueryRow("SELECT railway_token FROM setup_config WHERE id = 'default'").Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get railway token: %w", err)
	}
	if !token.Valid {
		return "", nil
	}
	return token.String, nil
}
