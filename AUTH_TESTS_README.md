# OAuth Authentication E2E Test Documentation

## Overview
The `/mnt/e/200Project/400SingleProject/20250925_randompic/randompic/frontend/tests/e2e/auth.spec.ts` file implements comprehensive E2E tests for Google OAuth authentication flow as part of Task T022.

## Test Coverage

### Core OAuth Authentication Flow (12 test scenarios × 5 browsers = 60 total tests)

1. **User can login with Google OAuth**
   - Mocks Google OAuth authorization flow
   - Tests complete redirect cycle: login → Google → callback → authenticated
   - Verifies authentication state indicators (user profile, logout button visible)

2. **User can access protected features after login**
   - Tests protected route access (/upload page)
   - Verifies user profile information display
   - Confirms authenticated user permissions

3. **Unauthenticated user is redirected from protected routes**
   - Tests access control for /upload page without authentication
   - Verifies redirect to login or home page
   - Ensures login button is visible

4. **Authentication state persists across browser refresh**
   - Tests session persistence after page reload
   - Verifies continued access to protected routes
   - Confirms authentication state indicators remain

5. **User can logout and return to unauthenticated state**
   - Tests logout functionality and state cleanup
   - Verifies removal of authentication indicators
   - Tests loss of protected route access after logout

### OAuth Error Handling

6. **OAuth error handling - user cancels authorization**
   - Mocks `access_denied` OAuth error
   - Tests graceful error handling and user feedback
   - Verifies user remains unauthenticated

7. **OAuth error handling - invalid state parameter**
   - Tests security validation of OAuth state parameter
   - Verifies protection against CSRF attacks
   - Ensures appropriate error messaging

8. **OAuth flow handles server errors gracefully**
   - Mocks 500 server error during OAuth callback
   - Tests error resilience and user communication
   - Verifies fallback to unauthenticated state

### Advanced Authentication Features

9. **Login button shows loading state during OAuth flow**
   - Tests UI feedback during authentication process
   - Verifies loading indicators and user experience
   - Ensures eventual completion of OAuth flow

10. **Authentication works across different browser tabs**
    - Tests session sharing between browser tabs
    - Verifies consistent authentication state
    - Confirms protected route access in all tabs

### Edge Case Handling

11. **Handles expired tokens gracefully**
    - Mocks token expiration and refresh failure
    - Tests automatic logout or re-authentication
    - Verifies graceful degradation of functionality

12. **Handles network errors during authentication**
    - Mocks network failures during OAuth flow
    - Tests error resilience and user feedback
    - Ensures graceful fallback behavior

## Mock OAuth Provider Configuration

The tests use a comprehensive mocking system that simulates:

- **Google OAuth Authorization Server**: `https://accounts.google.com/oauth/v2/auth`
- **OAuth Callback Handling**: `/auth/callback` endpoint
- **User Profile API**: `/auth/me` endpoint
- **Token Management**: Access and refresh token lifecycle

### Mock User Data
```typescript
{
  id: 'test-user-123',
  email: 'test@example.com',
  display_name: 'Test User',
  avatar_url: 'https://example.com/avatar.jpg',
  role: 'user',
  email_verified: true
}
```

## Test Data Attributes

The tests expect these data-testid attributes in the frontend:

### Authentication UI Elements
- `[data-testid="login-button"]` - Login button
- `[data-testid="logout-button"]` - Logout button
- `[data-testid="user-profile"]` - User profile display
- `[data-testid="user-menu"]` - User menu dropdown
- `[data-testid="user-email"]` - User email display

### Status and Feedback Elements
- `[data-testid="auth-success"]` - Authentication success message
- `[data-testid="auth-error"]` - Authentication error message
- `[data-testid="auth-loading"]` - Authentication loading indicator

### Protected Features
- `[data-testid="upload-button"]` - Upload feature button
- `[data-testid="upload-form"]` - Upload form (on /upload page)

## Browser Coverage

Tests run on 5 browser configurations:
- **Desktop**: Chromium, Firefox, WebKit
- **Mobile**: Chrome (Pixel 5), Safari (iPhone 12)

This ensures OAuth flow works consistently across all major browser engines and device types.

## Expected Test Behavior (TDD Phase)

**All tests should currently FAIL** because:
1. Authentication UI components are not implemented
2. OAuth endpoints are not configured
3. Protected routes are not set up
4. Authentication state management is missing

This is the expected behavior in the TDD phase - tests are written first and should fail until the implementation is complete.

## Integration Points

The tests integrate with:
- **Kinde OAuth 2.0 Provider** (when implemented)
- **Backend API authentication endpoints** (/auth/login, /auth/callback, /auth/logout)
- **Protected API endpoints** (authenticated image uploads, user profile)
- **Frontend authentication state management** (session persistence, UI updates)

## Performance Considerations

- Tests include 10-second timeouts for OAuth flows
- Network idle waits ensure complete page loads
- Cross-tab testing validates session performance
- Error scenarios test resilience and recovery time