# Capacity Update Form - Frontend Testing Guide

## Overview
This guide provides step-by-step instructions for testing the capacity update form that providers use to update hospital capacity in real-time.

**Related Documentation:**
- **Implementation Details:** See [`CAPACITY_UPDATE_IMPLEMENTATION.md`](./CAPACITY_UPDATE_IMPLEMENTATION.md)
- **Ward-Level Capacity:** See [`WARD_LEVEL_CAPACITY_IMPLEMENTATION.md`](./WARD_LEVEL_CAPACITY_IMPLEMENTATION.md)
- **Complete End-to-End Testing:** See [`PRODUCTION_LIKE_TESTING_GUIDE.md`](./PRODUCTION_LIKE_TESTING_GUIDE.md)

## 🚀 Quick Start (Easiest Method)

**Generate a test token and get the form URL in one command:**

```bash
cd backend
source ~/.nvm/nvm.sh && nvm use 24
npx ts-node scripts/test-form-manually.ts facility-test-1
```

This will:
- ✅ Start a test server automatically
- ✅ Create a test facility
- ✅ Generate a token
- ✅ Display the form URL
- ✅ Keep server running for testing

**Then:**
1. Copy the form URL from the output
2. Open it in your browser
3. Test the form!

---

## Prerequisites

1. **Provider API Server Running**
   - Default port: 3001 (or check `PROVIDER_PUBLIC_BASE_URL`)
   - Server must be accessible at the configured base URL

2. **Test Facility Created**
   - A facility profile must exist in MongoDB with an email address
   - Facility ID needed for generating test tokens

3. **Admin Token Configured**
   - Set `PROVIDER_ADMIN_TOKEN` environment variable
   - Used to trigger capacity requests

---

## Quick Start: Accessing the Form

### Easiest Method: Use the Test Script ⭐

The easiest way to generate a token and get the form URL:

```bash
cd backend
source ~/.nvm/nvm.sh && nvm use 24

# Run the test script (it starts its own server)
npx ts-node scripts/test-form-manually.ts facility-test-1
```

This script will:
1. ✅ Start a test server automatically
2. ✅ Create a test facility (if needed)
3. ✅ Generate a capacity request token
4. ✅ Display the form URL
5. ✅ Optionally open it in your browser (if `open` package installed)
6. ✅ Keep the server running so you can test

**Output:**
```
✅ Token generated successfully!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 CAPACITY UPDATE FORM URL:

   http://localhost:XXXXX/api/v1/capacity/form/abc123xyz...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🌐 To test the form:
   1. Copy the URL above
   2. Open it in your browser
   3. Fill out and submit the form
```

**Note:** The script starts its own test server on a random port. Press `Ctrl+C` to stop it.

### Alternative Method: Manual Token Generation (If Provider API is Running)

### Step 1: Start the Provider API Server

**Option A: Use the Test Script (Recommended)**
The test script (`scripts/test-form-manually.ts`) starts its own server, so you don't need to start the provider API separately.

**Option B: Start Provider API Manually**

If you want to test with the actual provider API server:

```bash
cd backend
source ~/.nvm/nvm.sh && nvm use 24

# Set required environment variables
export PROVIDER_ADMIN_TOKEN=test-admin-token
export PROVIDER_PUBLIC_BASE_URL=http://localhost:3001

# Start the server
npm run dev
# OR
npm start
```

The server should start on port 3001 (or your configured port).

### Step 2: Generate a Test Token

You need to generate a capacity request token for a test facility. You can do this via:

**Option A: Using the Admin API Endpoint**

```bash
# Replace 'facility-test-1' with your actual facility ID
curl -X POST http://localhost:3001/api/v1/capacity/request \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-admin-token" \
  -d '{
    "facilityId": "facility-test-1",
    "channel": "email"
  }'
```

**Option B: Using the Test Script**

Create a test facility and generate a token:

```bash
# Create a simple script to generate token
node -e "
const { DataProviderAPI } = require('./api/server');
const { FacilityProfileService } = require('./ingestion/facilityProfileService');
const { CapacityRequestService } = require('./ingestion/capacityRequestService');
// ... setup code
"
```

**Option C: Check Email/Logs**

If you've configured email sending, check the email for the form link. The link format is:
```
http://localhost:3001/api/v1/capacity/form/{token}
```

### Step 3: Access the Form in Browser

Once you have a token, open the form URL in your browser:

```
http://localhost:3001/api/v1/capacity/form/{your-token-here}
```

**Example:**
```
http://localhost:3001/api/v1/capacity/form/abc123xyz789...
```

---

## Testing Checklist

### 1. Visual Appearance Testing

#### Desktop Browser Testing
- [ ] **Chrome** - Open form and verify:
  - [ ] Form renders correctly
  - [ ] Colors match design system (#030213 primary, #f3f3f5 background)
  - [ ] Typography is clear and readable
  - [ ] Form fields are properly aligned
  - [ ] Button styling is correct
  - [ ] Container has proper padding and spacing

- [ ] **Firefox** - Verify same as Chrome
- [ ] **Safari** - Verify same as Chrome
- [ ] **Edge** - Verify same as Chrome

#### Mobile Device Testing
- [ ] **iOS Safari** - Test on iPhone:
  - [ ] Form is responsive and fits screen
  - [ ] Touch targets are large enough (min 44x44px)
  - [ ] Form fields are easy to tap
  - [ ] Keyboard appears correctly for number input
  - [ ] Form doesn't overflow horizontally

- [ ] **Android Chrome** - Test on Android device:
  - [ ] Same checks as iOS Safari
  - [ ] Verify on different screen sizes (small, medium, large)

#### Responsive Design Testing
Test at different viewport widths:
- [ ] **320px** (small mobile) - Form should be readable and usable
- [ ] **768px** (tablet) - Form should utilize space well
- [ ] **1024px** (small desktop) - Form should be centered with max-width
- [ ] **1920px** (large desktop) - Form should remain centered, not stretch

#### Browser Zoom Testing
- [ ] **50% zoom** - Form remains usable
- [ ] **100% zoom** (default) - Form looks correct
- [ ] **200% zoom** - Form remains readable and functional

---

### 2. Form Functionality Testing

#### Form Fields
- [ ] **Capacity Status Dropdown**
  - [ ] Dropdown shows all options: Available, Busy, Full, Closed
  - [ ] Can select each option
  - [ ] Selected value is visible
  - [ ] Dropdown is required (form won't submit without selection)

- [ ] **Average Wait Time Input**
  - [ ] Number input accepts only numbers
  - [ ] Can enter positive numbers (0, 1, 30, 100, etc.)
  - [ ] Negative numbers are prevented or handled
  - [ ] Decimal numbers work (if needed)
  - [ ] Field is optional (form can submit without it)

- [ ] **Urgent Care Available Checkbox**
  - [ ] Checkbox can be checked/unchecked
  - [ ] Checkbox state is visually clear
  - [ ] Field is optional

#### Form Submission
- [ ] **Valid Submission**
  - [ ] Fill all required fields
  - [ ] Click "Submit Update" button
  - [ ] Form submits successfully
  - [ ] Success page displays: "Thank You - Your capacity update has been recorded successfully"
  - [ ] Success page has proper styling

- [ ] **Invalid Submission**
  - [ ] Try to submit without capacity status
  - [ ] Verify form validation prevents submission
  - [ ] Error message is displayed (if client-side validation exists)

- [ ] **Invalid Capacity Status**
  - [ ] Try to submit with invalid status (if possible via dev tools)
  - [ ] Verify server returns error page
  - [ ] Error page shows valid options
  - [ ] Error page has link back to form

---

### 3. Accessibility Testing

#### Keyboard Navigation
- [ ] **Tab Navigation**
  - [ ] Can tab through all form fields in logical order
  - [ ] Focus indicator is visible (focus ring/outline)
  - [ ] Can activate dropdown with keyboard (Space/Enter)
  - [ ] Can check checkbox with keyboard (Space)
  - [ ] Can submit form with Enter key when button is focused

- [ ] **Screen Reader Testing**
  - [ ] Use VoiceOver (Mac) or NVDA (Windows)
  - [ ] Form fields are announced correctly
  - [ ] Labels are associated with inputs
  - [ ] Required fields are indicated
  - [ ] Error messages are announced
  - [ ] Success message is announced

#### ARIA Labels
- [ ] Form has proper ARIA labels
- [ ] Required fields are marked with `aria-required`
- [ ] Error messages are associated with fields via `aria-describedby`
- [ ] Form has proper `role` attributes

#### Color Contrast
- [ ] Text meets WCAG AA contrast requirements (4.5:1 for normal text)
- [ ] Primary button text is readable on dark background
- [ ] Form labels are readable
- [ ] Help text is readable

---

### 4. Error Handling Testing

#### Invalid Token Scenarios
- [ ] **Expired Token**
  - [ ] Generate token with short TTL (1 minute)
  - [ ] Wait for token to expire
  - [ ] Try to access form
  - [ ] Verify error message: "Token expired"
  - [ ] Error page is user-friendly

- [ ] **Invalid Token**
  - [ ] Access form with random/invalid token
  - [ ] Verify error message: "Invalid token"
  - [ ] Error page is user-friendly

- [ ] **Used Token**
  - [ ] Submit form successfully
  - [ ] Try to access form again with same token
  - [ ] Verify error message: "Token already used"
  - [ ] Error page is user-friendly

#### Form Validation Errors
- [ ] **Missing Required Field**
  - [ ] Try to submit without capacity status
  - [ ] Verify validation error (client or server-side)

- [ ] **Invalid Capacity Status**
  - [ ] Submit with invalid status value
  - [ ] Verify server returns 400 error
  - [ ] Error page lists valid options
  - [ ] Error page has link back to form

---

### 5. Cross-Browser Compatibility

Test the form in:
- [ ] **Chrome** (latest)
- [ ] **Firefox** (latest)
- [ ] **Safari** (latest)
- [ ] **Edge** (latest)
- [ ] **Mobile Safari** (iOS)
- [ ] **Mobile Chrome** (Android)

**Things to Check:**
- Form renders correctly in all browsers
- Styling is consistent
- Form submission works
- Error handling works
- Focus states work

---

### 6. Performance Testing

- [ ] **Page Load Time**
  - [ ] Form loads quickly (< 1 second on good connection)
  - [ ] No layout shift during load
  - [ ] Images/assets load correctly (if any)

- [ ] **Form Interaction**
  - [ ] Form fields respond immediately to input
  - [ ] Dropdown opens quickly
  - [ ] Form submission is responsive
  - [ ] No lag when typing in number field

---

## Testing Tools & Methods

### Browser DevTools

**Chrome DevTools:**
1. Open DevTools (F12 or Cmd+Option+I)
2. Use **Device Toolbar** (Cmd+Shift+M) to test responsive design
3. Use **Lighthouse** for accessibility audit
4. Use **Network** tab to monitor form submission
5. Use **Console** to check for JavaScript errors

**Testing Responsive Design:**
```
1. Open DevTools
2. Click Device Toolbar icon
3. Select different device presets:
   - iPhone SE (375px)
   - iPhone 12 Pro (390px)
   - iPad (768px)
   - Desktop (1920px)
4. Test form at each size
```

### Accessibility Testing Tools

**Browser Extensions:**
- **axe DevTools** (Chrome/Firefox) - Automated accessibility testing
- **WAVE** (Chrome/Firefox) - Web accessibility evaluation
- **Lighthouse** (built into Chrome) - Accessibility audit

**Screen Readers:**
- **VoiceOver** (Mac) - Built-in, activate with Cmd+F5
- **NVDA** (Windows) - Free, download from nvaccess.org
- **JAWS** (Windows) - Commercial option

### Manual Testing Script

Create a test script to generate tokens and test the form:

```bash
#!/bin/bash
# test-capacity-form.sh

BASE_URL="http://localhost:3001"
ADMIN_TOKEN="test-admin-token"
FACILITY_ID="facility-test-1"

# Generate token
echo "Generating capacity request token..."
RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/capacity/request" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -d "{\"facilityId\": \"${FACILITY_ID}\", \"channel\": \"email\"}")

echo "Response: ${RESPONSE}"

# Extract token from email (if using mock email sender)
# Or check server logs for the token
echo ""
echo "To test the form:"
echo "1. Check server logs or email for the form link"
echo "2. Open the link in your browser"
echo "3. Fill out and submit the form"
```

---

## Step-by-Step Testing Workflow

### Complete Test Flow

1. **Setup**
   ```bash
   cd backend
   source ~/.nvm/nvm.sh && nvm use 24
   export PROVIDER_ADMIN_TOKEN=test-admin-token
   export PROVIDER_PUBLIC_BASE_URL=http://localhost:3001
   npm run dev
   ```

2. **Create Test Facility** (if needed)
   - Use MongoDB or in-memory store
   - Ensure facility has email address

3. **Generate Test Token**
   ```bash
   curl -X POST http://localhost:3001/api/v1/capacity/request \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer test-admin-token" \
     -d '{"facilityId": "your-facility-id", "channel": "email"}'
   ```

4. **Extract Token from Response/Logs**
   - Check server console output
   - Or check email if configured
   - Token will be in the form link

5. **Open Form in Browser**
   ```
   http://localhost:3001/api/v1/capacity/form/{token}
   ```

6. **Test Form**
   - Fill out form fields
   - Submit form
   - Verify success page
   - Check backend logs for webhook trigger

7. **Test Error Scenarios**
   - Try expired token
   - Try invalid token
   - Try submitting invalid data

---

## Automated Visual Testing (Optional)

### Using Playwright or Cypress

You can automate some visual testing:

```typescript
// Example Playwright test
import { test, expect } from '@playwright/test';

test('capacity form renders correctly', async ({ page }) => {
  const token = 'your-test-token';
  await page.goto(`http://localhost:3001/api/v1/capacity/form/${token}`);
  
  // Check form elements
  await expect(page.locator('h1')).toContainText('Facility Capacity Update');
  await expect(page.locator('select[name="capacityStatus"]')).toBeVisible();
  await expect(page.locator('input[name="avgWaitMinutes"]')).toBeVisible();
  await expect(page.locator('button[type="submit"]')).toBeVisible();
  
  // Test form submission
  await page.selectOption('select[name="capacityStatus"]', 'busy');
  await page.fill('input[name="avgWaitMinutes"]', '45');
  await page.check('input[name="urgentCareAvailable"]');
  await page.click('button[type="submit"]');
  
  // Verify success page
  await expect(page.locator('text=Thank You')).toBeVisible();
});
```

---

## Common Issues & Troubleshooting

### Form Not Loading
- **Check server is running**: `curl http://localhost:3001/health`
- **Check token is valid**: Verify token hasn't expired
- **Check CORS**: Ensure CORS is enabled for your domain
- **Check browser console**: Look for JavaScript errors

### Form Submission Fails
- **Check network tab**: See what error is returned
- **Check server logs**: Look for error messages
- **Verify token**: Token might be expired or already used
- **Check validation**: Ensure capacity status is valid

### Styling Issues
- **Clear browser cache**: Hard refresh (Cmd+Shift+R / Ctrl+Shift+R)
- **Check CSS**: Verify styles are loading
- **Check viewport meta tag**: Should be present in HTML

---

## Test Data

### Sample Test Tokens

You can generate test tokens using the integration test:

```bash
cd backend
source ~/.nvm/nvm.sh && nvm use 24

# Run test to generate token (check console output)
npx ts-node tests/integration/provider_capacity_integration_test.ts
```

### Sample Facility IDs

Use these for testing (create them first):
- `facility-test-1`
- `facility-test-2`
- `facility-ttl-global`
- `facility-ttl-override`

---

## Reporting Test Results

Document your findings:

1. **Browser/Device**: Chrome 120 on macOS
2. **Screen Size**: 1920x1080
3. **Issues Found**: List any problems
4. **Screenshots**: Attach screenshots of issues
5. **Steps to Reproduce**: How to see the issue

---

## Next Steps After Testing

1. **Document Issues**: Create issues/tickets for any problems found
2. **Fix Critical Issues**: Address blocking issues first
3. **Retest**: Verify fixes work
4. **Deploy to Staging**: Test with real email delivery
5. **User Acceptance Testing**: Have providers test the form

---

## Quick Reference

**Form URL Format:**
```
{PROVIDER_PUBLIC_BASE_URL}/api/v1/capacity/form/{token}
```

**Default Server:**
```
http://localhost:3001
```

**Admin Endpoint:**
```
POST /api/v1/capacity/request
Authorization: Bearer {PROVIDER_ADMIN_TOKEN}
Body: { "facilityId": "...", "channel": "email" }
```

**Form Endpoint:**
```
GET /api/v1/capacity/form/:token
```

**Submit Endpoint:**
```
POST /api/v1/capacity/submit
Body: {
  "token": "...",
  "capacityStatus": "available|busy|full|closed",
  "avgWaitMinutes": 30,
  "urgentCareAvailable": true
}
```
