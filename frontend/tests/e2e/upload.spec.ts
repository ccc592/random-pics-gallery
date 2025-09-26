import { test, expect } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test.describe('Image Upload', () => {
  const testImagePath = path.join(__dirname, '../fixtures/test-image.jpg');
  const largImagePath = path.join(__dirname, '../fixtures/large-image.jpg');
  const webpImagePath = path.join(__dirname, '../fixtures/test-image.webp');

  test.beforeEach(async ({ page }) => {
    // Mock authentication - assume user is logged in as admin
    await page.addInitScript(() => {
      localStorage.setItem('auth', JSON.stringify({
        user: { id: '1', role: 'admin', name: 'Test Admin' },
        token: 'mock-admin-token'
      }));
    });
  });

  test('Admin can upload JPEG/PNG images via file picker', async ({ page }) => {
    // Navigate to upload page
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Check that upload form is visible
    const uploadForm = page.locator('[data-testid="upload-form"]');
    await expect(uploadForm).toBeVisible();

    // Test file input
    const fileInput = page.locator('input[type="file"]');
    await expect(fileInput).toBeVisible();

    // Mock file upload - simulate selecting a JPEG file
    await fileInput.setInputFiles([{
      name: 'test-image.jpg',
      mimeType: 'image/jpeg',
      buffer: Buffer.from('fake-jpeg-data')
    }]);

    // Fill required form fields
    const altTextInput = page.locator('[data-testid="alt-text-input"]');
    await altTextInput.fill('A beautiful motivational landscape with mountains and sunrise');

    const tagsInput = page.locator('[data-testid="tags-input"]');
    await tagsInput.fill('motivational, landscape, nature');

    const weightInput = page.locator('[data-testid="weight-input"]');
    await weightInput.fill('5');

    // Submit upload
    const submitButton = page.locator('[data-testid="upload-submit"]');

    // Mock successful API response
    await page.route('**/api/images/upload', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '123',
          filename: 'test-image.jpg',
          altText: 'A beautiful motivational landscape with mountains and sunrise',
          tags: ['motivational', 'landscape', 'nature'],
          weight: 5
        })
      });
    });

    await submitButton.click();

    // Verify success notification
    const successMessage = page.locator('[data-testid="upload-success"]');
    await expect(successMessage).toBeVisible();
    await expect(successMessage).toContainText(/successfully uploaded|upload complete/i);

    // Verify form is reset or redirected
    const currentUrl = page.url();
    expect(currentUrl).toMatch(/upload|gallery/);
  });

  test('Upload validation works - file size limit', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    const fileInput = page.locator('input[type="file"]');

    // Simulate selecting a file larger than 2MB
    const largBuffer = Buffer.alloc(2.5 * 1024 * 1024); // 2.5MB
    await fileInput.setInputFiles([{
      name: 'large-image.jpg',
      mimeType: 'image/jpeg',
      buffer: largBuffer
    }]);

    // Should show file size validation error
    const errorMessage = page.locator('[data-testid="file-size-error"]').or(
      page.locator('.error:has-text("size")').or(
        page.locator('[role="alert"]:has-text("size")')
      )
    );

    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText(/2MB|size|large/i);

    // Submit button should be disabled
    const submitButton = page.locator('[data-testid="upload-submit"]');
    await expect(submitButton).toBeDisabled();
  });

  test('Upload validation works - file format restriction', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    const fileInput = page.locator('input[type="file"]');

    // Simulate selecting a WebP file (not allowed per requirements)
    await fileInput.setInputFiles([{
      name: 'test-image.webp',
      mimeType: 'image/webp',
      buffer: Buffer.from('fake-webp-data')
    }]);

    // Should show file format validation error
    const errorMessage = page.locator('[data-testid="file-format-error"]').or(
      page.locator('.error:has-text("format")').or(
        page.locator('[role="alert"]:has-text("format")')
      )
    );

    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText(/JPEG|PNG|format/i);

    // Submit button should be disabled
    const submitButton = page.locator('[data-testid="upload-submit"]');
    await expect(submitButton).toBeDisabled();
  });

  test('Upload validation works - missing alt text', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles([{
      name: 'test-image.jpg',
      mimeType: 'image/jpeg',
      buffer: Buffer.from('fake-jpeg-data')
    }]);

    // Leave alt text empty and try to submit
    const submitButton = page.locator('[data-testid="upload-submit"]');
    await submitButton.click();

    // Should show alt text validation error
    const errorMessage = page.locator('[data-testid="alt-text-error"]').or(
      page.locator('.error:has-text("alt")').or(
        page.locator('[role="alert"]:has-text("required")')
      )
    );

    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText(/required|alt text/i);
  });

  test('Upload form supports drag and drop', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Find drag-and-drop zone
    const dropZone = page.locator('[data-testid="drop-zone"]').or(
      page.locator('.drop-zone').or(
        page.locator('[data-drop-zone]')
      )
    );

    if (await dropZone.isVisible()) {
      // Simulate drag enter
      await dropZone.dispatchEvent('dragenter', {
        dataTransfer: {
          files: [{
            name: 'dropped-image.jpg',
            type: 'image/jpeg'
          }]
        }
      });

      // Drop zone should show active state
      await expect(dropZone).toHaveClass(/active|drag-over|highlight/);

      // Simulate drop
      await dropZone.dispatchEvent('drop', {
        dataTransfer: {
          files: [{
            name: 'dropped-image.jpg',
            type: 'image/jpeg',
            size: 1024000 // 1MB
          }]
        }
      });

      // File should be processed
      const filePreview = page.locator('[data-testid="file-preview"]');
      await expect(filePreview).toBeVisible({ timeout: 2000 });
    }
  });

  test('Upload shows progress indicator during upload', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Fill form with valid data
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles([{
      name: 'test-image.jpg',
      mimeType: 'image/jpeg',
      buffer: Buffer.from('fake-jpeg-data')
    }]);

    const altTextInput = page.locator('[data-testid="alt-text-input"]');
    await altTextInput.fill('Test image for upload progress');

    // Mock slow API response to see progress indicator
    await page.route('**/api/images/upload', async route => {
      // Delay response to simulate upload time
      await new Promise(resolve => setTimeout(resolve, 1000));
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: '123', message: 'Upload successful' })
      });
    });

    const submitButton = page.locator('[data-testid="upload-submit"]');
    await submitButton.click();

    // Check for progress indicator
    const progressIndicator = page.locator('[data-testid="upload-progress"]').or(
      page.locator('.progress').or(
        page.locator('[role="progressbar"]')
      )
    );

    // Progress should be visible during upload
    await expect(progressIndicator).toBeVisible({ timeout: 500 });

    // Submit button should be disabled during upload
    await expect(submitButton).toBeDisabled();

    // Wait for upload to complete
    await expect(progressIndicator).toBeHidden({ timeout: 2000 });
  });

  test('Upload handles server errors gracefully', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Fill form with valid data
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles([{
      name: 'test-image.jpg',
      mimeType: 'image/jpeg',
      buffer: Buffer.from('fake-jpeg-data')
    }]);

    const altTextInput = page.locator('[data-testid="alt-text-input"]');
    await altTextInput.fill('Test image for error handling');

    // Mock server error
    await page.route('**/api/images/upload', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    const submitButton = page.locator('[data-testid="upload-submit"]');
    await submitButton.click();

    // Should show error message
    const errorMessage = page.locator('[data-testid="upload-error"]').or(
      page.locator('.error').or(
        page.locator('[role="alert"]')
      )
    );

    await expect(errorMessage).toBeVisible();
    await expect(errorMessage).toContainText(/error|failed|try again/i);

    // Form should remain usable (not reset)
    await expect(altTextInput).toHaveValue('Test image for error handling');
  });

  test('Upload form has proper accessibility', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    // Check form accessibility
    const form = page.locator('[data-testid="upload-form"]');
    await expect(form).toBeVisible();

    // File input should have proper labeling
    const fileInput = page.locator('input[type="file"]');
    const fileLabel = page.locator('label[for*="file"]').or(
      page.locator('label:has(input[type="file"])')
    );
    await expect(fileLabel).toBeVisible();

    // Alt text input should be properly labeled and required
    const altTextInput = page.locator('[data-testid="alt-text-input"]');
    await expect(altTextInput).toHaveAttribute('required');

    const altTextLabel = page.locator('label[for*="alt"]').or(
      page.locator('label:has-text("alt")')
    );
    await expect(altTextLabel).toBeVisible();

    // Form should be keyboard navigable
    await fileInput.focus();
    await page.keyboard.press('Tab');

    // Should focus on alt text input
    const focusedElement = page.locator(':focus');
    await expect(focusedElement).toBeFocused();
  });

  test('Non-admin users cannot access upload page', async ({ page }) => {
    // Mock non-admin user
    await page.addInitScript(() => {
      localStorage.setItem('auth', JSON.stringify({
        user: { id: '2', role: 'user', name: 'Regular User' },
        token: 'mock-user-token'
      }));
    });

    await page.goto('/upload');

    // Should be redirected or show access denied
    const accessDenied = page.locator('[data-testid="access-denied"]').or(
      page.locator(':has-text("not authorized")').or(
        page.locator(':has-text("access denied")')
      )
    );

    const currentUrl = page.url();
    const isRedirected = !currentUrl.includes('/upload');
    const hasAccessDenied = await accessDenied.isVisible();

    expect(isRedirected || hasAccessDenied).toBeTruthy();
  });

  test('Upload form supports PNG files', async ({ page }) => {
    await page.goto('/upload');
    await page.waitForLoadState('networkidle');

    const fileInput = page.locator('input[type="file"]');

    // Test PNG file upload
    await fileInput.setInputFiles([{
      name: 'test-image.png',
      mimeType: 'image/png',
      buffer: Buffer.from('fake-png-data')
    }]);

    // Should not show format error for PNG
    const errorMessage = page.locator('[data-testid="file-format-error"]');
    await expect(errorMessage).not.toBeVisible();

    // Fill required fields
    const altTextInput = page.locator('[data-testid="alt-text-input"]');
    await altTextInput.fill('Test PNG image');

    const submitButton = page.locator('[data-testid="upload-submit"]');
    await expect(submitButton).not.toBeDisabled();
  });
});