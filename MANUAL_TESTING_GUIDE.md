# Manual Testing Guide - ujiscan Phase 9

## How to Test ujiscan Manually (Frontend + Backend)

### Prerequisites
- Go installed (backend)
- Browser (Chrome, Firefox, Safari)
- Terminal
- Basic curl knowledge (optional)

---

## STEP 1: Start the Backend

### Terminal 1: Start ujiscan server
```bash
cd /path/to/ujiscan
./ujiscan

# Expected output:
# ✅ SQLite database initialized: .../ujiscan/ujiscan.db
# ✅ Tool registry loaded successfully (5 tools, 4 modes)
# ℹ️  No saved engagement state found (first run)
# ujiscan server starting on http://localhost:8081
# API endpoints: [list of endpoints]
```

✅ **Server is ready when you see:** `ujiscan server starting on http://localhost:8081`

---

## STEP 2: Test Frontend in Browser

### Open Login Page
```
URL: http://localhost:8081/html/login.html
```

Expected:
- ☠️ ujiscan logo
- Email field (pre-filled: admin@ujiscan.local)
- Password field
- Sign In button
- Theme toggle (☀️/🌙) in top right
- Demo credentials section (shows email/password)

### Test 1: Login with Admin Credentials
```
Email:    admin@ujiscan.local
Password: admin123
Click "Sign In"
```

Expected:
- ✅ Login succeeds
- ✅ Page redirects to team dashboard
- ✅ JWT token stored in localStorage
- ✅ User name displayed in navbar

### Verify Token Storage
Open **Browser DevTools** (F12):
```
Console → Type:
  localStorage.getItem('ujiscan_token')
  localStorage.getItem('ujiscan_user')

Expected output:
  "token_XXXXXXXXXXXXX_admin"
  "{"email":"admin@ujiscan.local","role":"admin"}"
```

---

## STEP 3: Test Team Dashboard

### Page: Team Dashboard
```
URL: http://localhost:8081/html/dashboard-team.html
```

Expected:
- ✅ Navbar: ujiscan | admin@ujiscan.local (admin) | 🌙 | Logout
- ✅ Page title: "Team Dashboard"
- ✅ 5 metric cards (may show "-" if no data, that's ok):
  - Total Engagements
  - Active Scans
  - Critical Findings
  - High Findings
  - Resolved Issues
- ✅ "Recent Activity" section (may be empty)
- ✅ "Team Engagements" section (may be empty)

### Test: Dark Mode Toggle
```
Click 🌙 button in top right
```

Expected:
- ✅ Page switches to light mode
- ✅ Background becomes white
- ✅ Text becomes dark
- ✅ Icon changes to ☀️

```
Click ☀️ button again
```

Expected:
- ✅ Page switches back to dark mode
- ✅ Icon changes back to 🌙
- ✅ Theme persists after page reload

### Verify Theme Persistence
```
Click 🌙 to switch to light mode
Reload page (Ctrl+R or Cmd+R)
```

Expected:
- ✅ Page stays in light mode (theme persisted in localStorage)

---

## STEP 4: Test Client Dashboard

### Navigate to Client Dashboard
```
URL: http://localhost:8081/html/dashboard-client.html
```

Expected:
- ✅ Navbar same as before
- ✅ Page title: "Client Dashboard"
- ✅ "Overall Risk Score: -" (no data yet, that's ok)
- ✅ Color-coded risk display (will be "Low Risk" by default)
- ✅ Finding cards sections (Critical/High/Medium/Low may be empty)
- ✅ "Recommended Actions" section (may be empty)

---

## STEP 5: Test Admin Dashboard

### Navigate to Admin Dashboard
```
URL: http://localhost:8081/html/dashboard-admin.html
```

Expected:
- ✅ Navbar same as before
- ✅ Page title: "Admin Dashboard"
- ✅ 3 tabs visible: "System Health | User Management | Audit Logs"
- ✅ System Health tab is active by default

### Test: System Health Tab
```
Default view shows:
```

Expected metrics (even if values are "-" or "0"):
- API Status
- Database Status
- Total Users
- Active Sessions
- Total Engagements
- Audit Events
- Service Status list

### Test: User Management Tab
```
Click "User Management" tab
```

Expected:
- ✅ Table with columns: Email | Role | Status | Created
- ✅ At least admin user visible
- ✅ Can see role badge (admin)

### Test: Audit Logs Tab
```
Click "Audit Logs" tab
```

Expected:
- ✅ List of audit log entries (may be empty initially)
- ✅ Shows: User | Action | Timestamp

---

## STEP 6: Test Logout

### From any dashboard:
```
Click "Logout" button (top right)
```

Expected:
- ✅ Redirects to login page
- ✅ localStorage cleared (tokens removed)
- ✅ Must login again to access dashboards

### Verify:
Open DevTools Console:
```
localStorage.getItem('ujiscan_token')
```

Expected: `null` (token was cleared)

---

## STEP 7: Test API with curl (Optional)

### Terminal 2: Test API endpoints
```bash
# 1. Get admin token
TOKEN=$(curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@ujiscan.local","password":"admin123"}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Token: $TOKEN"

# 2. Test protected endpoint (requires token)
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/status | jq

# 3. Test engagements endpoint
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/engagements | jq

# 4. Test without token (should fail with 401)
curl -s http://localhost:8081/api/status
# Expected: "missing authorization header"
```

---

## STEP 8: Test Mobile Responsiveness

### In Browser DevTools (F12):
```
1. Click Device Toolbar icon (or Ctrl+Shift+M)
2. Select different devices:
   - iPhone SE
   - iPad
   - Galaxy S5
```

Expected:
- ✅ Login page is responsive (inputs stack vertically)
- ✅ Dashboard cards reflow (1 column on mobile, grid on desktop)
- ✅ Navbar collapses/adapts (user info visible)
- ✅ No horizontal scrolling
- ✅ All buttons clickable

---

## STEP 9: Test Error Scenarios

### Try wrong password:
```
URL: http://localhost:8081/html/login.html
Email: admin@ujiscan.local
Password: wrongpassword
Click "Sign In"
```

Expected:
- ✅ Error message: "Invalid email or password"
- ✅ Button re-enabled for retry
- ✅ Page does NOT redirect

### Try accessing dashboard without login:
```
1. Clear localStorage:
   Open DevTools → Storage → localStorage → Delete ujiscan_token
2. Navigate to: http://localhost:8081/html/dashboard-team.html
```

Expected:
- ✅ Redirects to login page
- ✅ Cannot access dashboard without token

---

## STEP 10: Test Network Requests

### Monitor API calls in Browser:

Open DevTools → **Network** tab:
```
1. Go to login page
2. Enter credentials
3. Click Sign In
4. Watch Network tab
```

Expected requests:
```
POST /auth/login
├─ Status: 200 OK
├─ Response: {"token":"...", "email":"...", "role":"..."}
└─ Check Headers: Content-Type: application/json
```

### Monitor dashboard requests:

```
1. Login successfully
2. Navigate to dashboard
3. Watch Network tab
```

Expected:
```
GET /api/status
GET /api/tools
GET /api/engagements
GET /api/audit/logs (may 404, that's ok for now)
...
(All show 200 OK or 404 for placeholder endpoints)
```

### Check Authorization Header:

Click on any API request in Network tab:
```
Request Headers →
  Authorization: Bearer token_XXXXX
```

Expected:
- ✅ Every API request includes Bearer token
- ✅ No token = 401 error

---

## Test Checklist

```
✅ Backend starts without errors
✅ Login page loads at /html/login.html
✅ Login succeeds with admin@ujiscan.local / admin123
✅ Token stored in localStorage
✅ Redirects to team dashboard
✅ Team dashboard loads and displays UI
✅ Client dashboard loads and displays UI
✅ Admin dashboard loads with all tabs
✅ Dark mode toggle works
✅ Theme persists after reload
✅ Logout clears token and redirects
✅ API calls include Authorization header
✅ Missing token returns 401
✅ Mobile responsive (iPhone/iPad)
✅ No console errors (check DevTools)
✅ All buttons are clickable
```

---

## Troubleshooting

### Backend won't start
```bash
# Check if port 8081 is already in use
lsof -i :8081

# Kill existing process
kill -9 <PID>

# Try again
./ujiscan
```

### Pages show blank/errors
```
1. Check browser console (F12 → Console tab)
2. Look for JavaScript errors
3. Check Network tab for failed requests
4. Verify backend is still running
```

### Token not working
```
1. Check localStorage has token: 
   DevTools → Storage → localStorage
2. Check token in Authorization header (Network tab)
3. Try logging out and logging in again
4. Clear browser cache/cookies and retry
```

### Theme not persisting
```
1. Check localStorage for ujiscan_theme
2. Try in private/incognito window
3. Check if localStorage is disabled
```

---

## What's Expected to FAIL

These are known limitations (marked for post-v1.0):

- ❌ Dashboard metric cards show "-" (data endpoints incomplete)
- ❌ Recent Activity list is empty (audit log hooks incomplete)
- ❌ Audit Logs tab is empty (needs backend work)
- ❌ Creating engagements shows 404 (endpoint not fully wired)

These are **OK for v1.0** - core login/auth works perfectly!

---

## Next Steps After Manual Testing

If everything works:
1. ✅ Phase 9 validation complete
2. ➜ Ready for Phase 10 (Docker)
3. ➜ Then v1.0 release (Nov 10)

If issues found:
1. Document the issue
2. Check PHASE_9_E2E_REPORT.md for known issues
3. Report if it's a new bug
4. Fix before Docker setup

---

Happy testing! 🧪✅

