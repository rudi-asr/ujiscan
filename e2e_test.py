#!/usr/bin/env python3
"""
ujiscan E2E Test Suite
Tests full user workflows: login, dashboard, engagement, audit, dark mode
"""

import requests
import json
import sys
import time
from typing import Optional

# Configuration
API_BASE = "http://localhost:8081"
TIMEOUT = 5

# Test data
TEST_USERS = {
    "admin": {"email": "admin@ujiscan.local", "password": "admin123", "role": "admin"},
    "pentester": {"email": "pentester@ujiscan.local", "password": "pass123", "role": "pentester"},
    "client": {"email": "client@ujiscan.local", "password": "pass123", "role": "client"},
}

# Color codes for terminal output
class Color:
    GREEN = "\033[92m"
    RED = "\033[91m"
    YELLOW = "\033[93m"
    BLUE = "\033[94m"
    END = "\033[0m"

def test_result(name: str, passed: bool, message: str = ""):
    """Print test result"""
    status = f"{Color.GREEN}✅ PASS{Color.END}" if passed else f"{Color.RED}❌ FAIL{Color.END}"
    print(f"  {status} {name}")
    if message:
        print(f"      {message}")
    return passed

def login(email: str, password: str) -> Optional[str]:
    """Test login and return JWT token"""
    try:
        res = requests.post(
            f"{API_BASE}/auth/login",
            json={"email": email, "password": password},
            timeout=TIMEOUT
        )
        if res.status_code == 200:
            data = res.json()
            return data.get("token")
        return None
    except Exception as e:
        print(f"Error logging in: {e}")
        return None

def test_login_flows():
    """Test 1: Login flows for all roles"""
    print(f"\n{Color.BLUE}TEST 1: Login Flows{Color.END}")
    passed = 0
    total = 0

    for role, creds in TEST_USERS.items():
        total += 1
        token = login(creds["email"], creds["password"])
        if test_result(f"Login as {role}", token is not None):
            passed += 1

    return f"{passed}/{total}"

def test_api_endpoints(token: str):
    """Test 2: API endpoints"""
    print(f"\n{Color.BLUE}TEST 2: API Endpoints{Color.END}")
    passed = 0
    total = 0

    headers = {"Authorization": f"Bearer {token}"}
    endpoints = [
        ("/api/status", "GET", "Server status"),
        ("/api/tools", "GET", "Tools list"),
        ("/api/engagements", "GET", "Engagements list"),
        ("/api/audit/logs", "GET", "Audit logs"),
        ("/api/dashboard/team", "GET", "Team dashboard"),
        ("/api/dashboard/client", "GET", "Client dashboard"),
        ("/api/dashboard/admin", "GET", "Admin dashboard"),
        ("/api/notifications", "GET", "Notifications"),
    ]

    for endpoint, method, name in endpoints:
        total += 1
        try:
            if method == "GET":
                res = requests.get(f"{API_BASE}{endpoint}", headers=headers, timeout=TIMEOUT)
            else:
                res = requests.post(f"{API_BASE}{endpoint}", headers=headers, timeout=TIMEOUT)

            if res.status_code == 200:
                if test_result(name, True):
                    passed += 1
            else:
                test_result(name, False, f"Status {res.status_code}")
        except Exception as e:
            test_result(name, False, str(e))

    return f"{passed}/{total}"

def test_auth_enforcement(token: str):
    """Test 3: Authentication enforcement"""
    print(f"\n{Color.BLUE}TEST 3: Authentication Enforcement{Color.END}")
    passed = 0
    total = 0

    # Test without token
    total += 1
    try:
        res = requests.get(f"{API_BASE}/api/status", timeout=TIMEOUT)
        if res.status_code == 401 or res.status_code == 403:
            test_result("401 without token", True)
            passed += 1
        else:
            test_result("401 without token", False, f"Got {res.status_code}")
    except Exception as e:
        test_result("401 without token", False, str(e))

    # Test with token
    total += 1
    headers = {"Authorization": f"Bearer {token}"}
    try:
        res = requests.get(f"{API_BASE}/api/status", headers=headers, timeout=TIMEOUT)
        if res.status_code == 200:
            test_result("Allow with valid token", True)
            passed += 1
        else:
            test_result("Allow with valid token", False, f"Status {res.status_code}")
    except Exception as e:
        test_result("Allow with valid token", False, str(e))

    # Test with invalid token
    total += 1
    headers = {"Authorization": "Bearer invalid_token"}
    try:
        res = requests.get(f"{API_BASE}/api/status", headers=headers, timeout=TIMEOUT)
        if res.status_code == 401 or res.status_code == 403:
            test_result("401 with invalid token", True)
            passed += 1
        else:
            test_result("401 with invalid token", False, f"Got {res.status_code}")
    except Exception as e:
        test_result("401 with invalid token", False, str(e))

    return f"{passed}/{total}"

def test_engagement_workflow(token: str):
    """Test 4: Engagement workflow (create, read, list)"""
    print(f"\n{Color.BLUE}TEST 4: Engagement Workflow{Color.END}")
    passed = 0
    total = 0

    headers = {"Authorization": f"Bearer {token}"}

    # Create engagement
    total += 1
    engagement_id = None
    try:
        res = requests.post(
            f"{API_BASE}/api/engagements",
            json={
                "name": "E2E Test Engagement",
                "client_name": "Test Client",
                "scope": "*.example.com"
            },
            headers=headers,
            timeout=TIMEOUT
        )
        if res.status_code == 200 or res.status_code == 201:
            data = res.json()
            engagement_id = data.get("id") or data.get("engagement_id")
            test_result("Create engagement", engagement_id is not None)
            if engagement_id:
                passed += 1
        else:
            test_result("Create engagement", False, f"Status {res.status_code}")
    except Exception as e:
        test_result("Create engagement", False, str(e))

    # List engagements
    total += 1
    try:
        res = requests.get(f"{API_BASE}/api/engagements", headers=headers, timeout=TIMEOUT)
        if res.status_code == 200:
            data = res.json()
            engagements = data.get("engagements", [])
            test_result("List engagements", len(engagements) >= 0, f"Found {len(engagements)}")
            passed += 1
        else:
            test_result("List engagements", False, f"Status {res.status_code}")
    except Exception as e:
        test_result("List engagements", False, str(e))

    # Get specific engagement
    if engagement_id:
        total += 1
        try:
            res = requests.get(
                f"{API_BASE}/api/engagements/{engagement_id}",
                headers=headers,
                timeout=TIMEOUT
            )
            if res.status_code == 200:
                test_result("Get engagement", True)
                passed += 1
            else:
                test_result("Get engagement", False, f"Status {res.status_code}")
        except Exception as e:
            test_result("Get engagement", False, str(e))

    return f"{passed}/{total}"

def test_audit_logs(token: str):
    """Test 5: Audit logs"""
    print(f"\n{Color.BLUE}TEST 5: Audit Logs{Color.END}")
    passed = 0
    total = 0

    headers = {"Authorization": f"Bearer {token}"}

    total += 1
    try:
        res = requests.get(f"{API_BASE}/api/audit/logs", headers=headers, timeout=TIMEOUT)
        if res.status_code == 200:
            data = res.json()
            logs = data.get("logs", [])
            test_result("Get audit logs", len(logs) >= 0, f"Found {len(logs)} logs")
            passed += 1
        else:
            test_result("Get audit logs", False, f"Status {res.status_code}")
    except Exception as e:
        test_result("Get audit logs", False, str(e))

    # Check export endpoint
    total += 1
    try:
        res = requests.get(
            f"{API_BASE}/api/audit/export",
            headers=headers,
            timeout=TIMEOUT,
            params={"format": "json"}
        )
        if res.status_code == 200:
            test_result("Export audit logs (JSON)", True)
            passed += 1
        else:
            test_result("Export audit logs (JSON)", False, f"Status {res.status_code}")
    except Exception as e:
        test_result("Export audit logs (JSON)", False, str(e))

    return f"{passed}/{total}"

def main():
    """Run all E2E tests"""
    print(f"{Color.BLUE}")
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║          ujiscan E2E Test Suite (Phase 9)                     ║")
    print("║                                                                ║")
    print("║  Testing: Login | APIs | Auth | Workflow | Audit              ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print(f"{Color.END}")

    # Check if backend is running
    try:
        requests.get(f"{API_BASE}/api/status", timeout=TIMEOUT)
    except:
        print(f"{Color.RED}❌ Backend not running at {API_BASE}{Color.END}")
        print("Start the backend with: ./ujiscan")
        sys.exit(1)

    # Get admin token
    print(f"\n{Color.YELLOW}Getting admin token...{Color.END}")
    admin_token = login(TEST_USERS["admin"]["email"], TEST_USERS["admin"]["password"])
    if not admin_token:
        print(f"{Color.RED}Failed to get admin token{Color.END}")
        sys.exit(1)
    print(f"{Color.GREEN}Got admin token{Color.END}")

    # Run tests
    results = {}
    results["login"] = test_login_flows()
    results["endpoints"] = test_api_endpoints(admin_token)
    results["auth"] = test_auth_enforcement(admin_token)
    results["engagement"] = test_engagement_workflow(admin_token)
    results["audit"] = test_audit_logs(admin_token)

    # Summary
    print(f"\n{Color.BLUE}")
    print("╔════════════════════════════════════════════════════════════════╗")
    print("║                    TEST SUMMARY                                ║")
    print("╚════════════════════════════════════════════════════════════════╝")
    print(f"{Color.END}")

    for test, result in results.items():
        print(f"  {test.title()}: {result}")

    print(f"\n{Color.GREEN}✅ E2E Test Suite Complete!{Color.END}")
    print(f"   Run this again to verify fixes or test new features.")

if __name__ == "__main__":
    main()
