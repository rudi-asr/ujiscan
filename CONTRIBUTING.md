# Contributing to ujiscan

Thank you for considering contributing to ujiscan! This document provides guidelines for contributing.

## Code of Conduct

Be respectful, inclusive, and professional. We don't tolerate harassment or discrimination.

## How Can I Contribute?

### Reporting Bugs

Before submitting a bug report, check existing issues. When creating a new issue, include:

- **Description:** Clear explanation of the problem
- **Steps to reproduce:** Exact steps to trigger the bug
- **Expected behavior:** What should happen
- **Actual behavior:** What actually happens
- **Environment:** OS, Go version, Docker version
- **Logs:** Output from `docker logs ujiscan-container` or terminal

**Example:**
```
Title: Login button unresponsive on Safari

Description:
When clicking the "Sign In" button on the login page, nothing happens.

Steps to Reproduce:
1. Open http://localhost:8081/html/login.html in Safari 18.0
2. Enter admin@ujiscan.local / admin123
3. Click "Sign In" button
4. No response

Environment:
- macOS 14.7
- Safari 18.0
- ujiscan v1.0.0

Logs:
[No errors in console]
```

### Suggesting Enhancements

Submit feature requests as GitHub Discussions. Include:

- **Use case:** Why is this feature needed?
- **Proposed solution:** How should it work?
- **Alternatives:** Other approaches considered
- **Impact:** Which users/roles benefit?

**Example:**
```
Title: Multi-language support (i18n)

Use case:
International teams need ujiscan in multiple languages. Currently, all UI text is English.

Proposed solution:
1. Add i18n library (e.g., go-i18n)
2. Extract strings to YAML config
3. Add language selector in UI

Impact:
- Team users worldwide
- Increases adoption in non-English markets
```

### Submitting Pull Requests

1. **Fork the repository**
   ```bash
   git clone https://github.com/YOUR_USERNAME/ujiscan.git
   cd ujiscan
   git checkout -b feature/your-feature
   ```

2. **Make your changes**
   - Follow the code style (see below)
   - Write clear commit messages
   - Include tests if applicable

3. **Test your changes**
   ```bash
   go test ./...
   python3 e2e_test.py
   docker build -t ujiscan:dev .
   docker run -p 8081:8081 ujiscan:dev
   ```

4. **Push and create PR**
   ```bash
   git push origin feature/your-feature
   ```
   Then create a PR on GitHub with:
   - Clear description of changes
   - Reference any related issues
   - Screenshots (if UI changes)
   - Test results

## Development Setup

### Prerequisites
- Go 1.23+
- Docker & docker-compose
- Python 3.8+ (for testing)
- Git

### Local Development

```bash
# Clone repository
git clone https://github.com/rudi-asr/ujiscan.git
cd ujiscan

# Build binary
go build -o ujiscan .

# Run server
./ujiscan

# In another terminal, run tests
python3 e2e_test.py
```

### Project Structure

```
ujiscan/
├── cmd/server/          # Entry point
├── internal/
│   ├── auth/            # Authentication & RBAC
│   ├── engagement/      # Engagement management
│   ├── findings/        # Findings & evidence
│   ├── tools/           # Tool registry
│   ├── playbooks/       # Playbook orchestration
│   ├── reports/         # Report generation
│   ├── audit/           # Audit logging
│   ├── db/              # Database schema
│   └── persistence/     # State persistence
├── web/                 # Frontend (HTML/JS/CSS)
├── config/              # YAML configurations
└── e2e_test.py         # Automated tests
```

### Adding a New Feature

1. **Create a package under `internal/`**
   ```bash
   mkdir -p internal/myfeature
   touch internal/myfeature/types.go
   touch internal/myfeature/store.go
   touch internal/myfeature/handlers.go
   ```

2. **Implement types, store, and handlers**
   ```go
   // internal/myfeature/types.go
   type MyFeature struct {
       ID        string
       Name      string
       CreatedAt int64
   }

   // internal/myfeature/store.go
   type Store struct {
       items map[string]*MyFeature
   }

   func (s *Store) Create(f *MyFeature) error { ... }
   func (s *Store) Get(id string) (*MyFeature, error) { ... }
   func (s *Store) List() ([]*MyFeature, error) { ... }

   // internal/myfeature/handlers.go
   func (s *Store) HandleCreate(w http.ResponseWriter, r *http.Request) { ... }
   ```

3. **Register routes in main.go**
   ```go
   myFeatureStore := myfeature.NewStore()
   mux.HandleFunc("POST /api/myfeature", auth.RequireAuth(myFeatureStore.HandleCreate))
   ```

4. **Add tests**
   ```bash
   # Create test file
   touch internal/myfeature/handlers_test.go

   # Run tests
   go test ./internal/myfeature/...
   ```

5. **Update E2E tests** (`e2e_test.py`)
   ```python
   def test_myfeature():
       resp = post("/api/myfeature", {"name": "test"})
       assert resp.status_code == 201
   ```

## Code Style

### Go

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Name exported functions with verbs (e.g., `CreateUser`, `GetFinding`)
- Write comments for exported types and functions

**Example:**
```go
// User represents a system user.
type User struct {
    ID    string
    Email string
}

// CreateUser creates a new user.
func (s *Store) CreateUser(u *User) error {
    if u.Email == "" {
        return errors.New("email required")
    }
    s.users[u.ID] = u
    return nil
}
```

### JavaScript

- Use Vanilla JS (no frameworks)
- Use `const` by default, `let` if reassignment needed
- Avoid globals; use modules/closures
- Add JSDoc comments for functions

**Example:**
```javascript
/**
 * Authenticate user with email and password
 * @param {string} email - User email
 * @param {string} password - User password
 * @returns {Promise<Object>} - User object with token
 */
const authenticate = async (email, password) => {
    const response = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
    });
    if (!response.ok) throw new Error('Login failed');
    return response.json();
};
```

### HTML/CSS

- Use semantic HTML5 (`<nav>`, `<article>`, etc.)
- Dark mode by default (`data-theme="dark"`)
- Use CSS custom properties for colors
- Mobile-first responsive design

**Example:**
```html
<article class="card">
    <h2>Engagement Title</h2>
    <p class="status active">In Progress</p>
</article>

<style>
:root {
    --bg-primary: #1a1a1a;
    --text-primary: #ffffff;
}

html[data-theme="light"] {
    --bg-primary: #ffffff;
    --text-primary: #000000;
}

.card {
    background: var(--bg-primary);
    color: var(--text-primary);
}
</style>
```

## Commit Messages

Use clear, descriptive commit messages:

```
type: brief description

Detailed explanation of changes, if needed.

Related to: #123 (GitHub issue)
```

**Types:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `style:` Code style (formatting, whitespace)
- `refactor:` Code refactoring
- `test:` Adding/updating tests
- `chore:` Maintenance

**Examples:**
```
feat: Add multi-user support to dashboards

- Create users table in SQLite
- Add user store and handlers
- Update authentication middleware
- Add unit tests for user endpoints

Related to: #42

---

fix: Handle missing CORS header in login

Previously, login requests from different origins were rejected.
Now properly return CORS headers on all endpoints.

Related to: #38

---

docs: Update README with AWS deployment instructions
```

## Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
# Start server
./ujiscan &

# Run E2E tests
python3 e2e_test.py

# Kill server
pkill ujiscan
```

### Docker Testing

```bash
docker build -t ujiscan:test .
docker run -p 8081:8081 ujiscan:test

# In another terminal
curl http://localhost:8081/api/status
```

## Documentation

For significant changes, update relevant documentation:

- **README.md** - User-facing features
- **DEPLOYMENT_GUIDE.md** - Deployment/operations
- **PHASE_8B_HANDOFF.md** - Architecture, API design

Use Markdown with clear examples.

## Review Process

1. **Automated checks**
   - Tests pass (`go test ./...`)
   - Go vet passes (`go vet ./...`)
   - Docker builds successfully

2. **Code review**
   - At least 1 approval from maintainer
   - No conflicts with main branch
   - Follows code style guidelines

3. **Merge**
   - Squash commits if many
   - Update CHANGELOG.md
   - Delete feature branch

## License

By contributing, you agree that your contributions will be licensed under the same AGPL-3.0 license.

## Questions?

- Open a GitHub Discussion
- Email: asruddin@ujiscan.local
- Check existing documentation

---

**Thank you for contributing to ujiscan! 🙏🚀**
