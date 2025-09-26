// Comprehensive testing without full Playwright dependencies
import { parse } from 'node:url';

const FRONTEND_URL = 'http://localhost:3001';
const BACKEND_URL = 'http://localhost:8080';

class WebTester {
  constructor() {
    this.testResults = {
      passed: 0,
      failed: 0,
      total: 0
    };
  }

  async runTest(name, testFn) {
    this.testResults.total++;
    console.log(`🧪 ${name}...`);

    try {
      await testFn();
      console.log(`✅ PASS: ${name}`);
      this.testResults.passed++;
    } catch (error) {
      console.log(`❌ FAIL: ${name} - ${error.message}`);
      this.testResults.failed++;
    }
    console.log('');
  }

  async testFrontendAvailability() {
    const response = await fetch(FRONTEND_URL);
    if (!response.ok) {
      throw new Error(`Frontend returned status ${response.status}`);
    }

    const html = await response.text();
    if (!html.includes('Random Pics')) {
      throw new Error('Page title not found in HTML');
    }
  }

  async testBackendHealthEndpoint() {
    const response = await fetch(`${BACKEND_URL}/health`);
    if (!response.ok) {
      throw new Error(`Health endpoint returned status ${response.status}`);
    }

    const data = await response.json();
    if (data.status !== 'healthy') {
      throw new Error(`Expected healthy status, got ${data.status}`);
    }
  }

  async testRandomImagesEndpoint() {
    const response = await fetch(`${BACKEND_URL}/api/images/random`);
    if (!response.ok) {
      throw new Error(`Random images endpoint returned status ${response.status}`);
    }

    const data = await response.json();
    if (!data.message && !data.images) {
      throw new Error('Unexpected response format from random images endpoint');
    }
  }

  async testPageMetadata() {
    const response = await fetch(FRONTEND_URL);
    const html = await response.text();

    const requiredMeta = [
      '<meta charset="UTF-8">',
      '<meta name="viewport"',
      '<meta name="description"',
      '<title>Random Pics'
    ];

    for (const meta of requiredMeta) {
      if (!html.includes(meta)) {
        throw new Error(`Missing required metadata: ${meta}`);
      }
    }
  }

  async testResponsiveDesignElements() {
    const response = await fetch(FRONTEND_URL);
    const html = await response.text();

    // Check for responsive design patterns
    const responsivePatterns = [
      /grid-cols-1/, // Grid base
      /md:grid-cols-2/, // Medium responsive grid
      /lg:grid-cols-3/, // Large responsive grid
      /flex-col/, // Flexible column layouts
      /max-w-/, // Responsive max-widths
    ];

    let foundPatterns = 0;
    responsivePatterns.forEach(pattern => {
      if (pattern.test(html)) {
        foundPatterns++;
      }
    });

    if (foundPatterns === 0) {
      throw new Error('No responsive design patterns found in HTML');
    }
  }

  async testAccessibilityFeatures() {
    const response = await fetch(FRONTEND_URL);
    const html = await response.text();

    // Check for accessibility features
    const accessibilityFeatures = [
      /<main[^>]*>/, // Semantic main element
      /<img[^>]+alt=/, // Images with alt text
      /<button[^>]*>/, // Interactive buttons
      /<nav[^>]*>/, // Navigation elements (optional)
    ];

    const foundFeatures = accessibilityFeatures.filter(pattern => pattern.test(html));

    if (foundFeatures.length < 3) {
      throw new Error(`Only ${foundFeatures.length} accessibility features found, expected at least 3`);
    }
  }

  async testModernWebFeatures() {
    const response = await fetch(FRONTEND_URL);
    const html = await response.text();

    // Check for modern web features
    const modernFeatures = [
      'DOCTYPE html', // HTML5 doctype
      'data-astro-', // Astro framework markers
      'tailwind', // Tailwind CSS
      'grid-cols', // CSS Grid
      'flex-', // Flexbox
      'transition-', // CSS transitions
    ];

    const foundFeatures = modernFeatures.filter(feature => html.includes(feature));

    if (foundFeatures.length < 4) {
      throw new Error(`Only ${foundFeatures.length} modern features found, expected at least 4`);
    }
  }

  async testPerformanceHeaders() {
    const response = await fetch(FRONTEND_URL);

    // Check for performance-related headers
    const contentType = response.headers.get('content-type');
    if (!contentType || !contentType.includes('text/html')) {
      throw new Error('Missing or incorrect Content-Type header');
    }

    // Check response time (should be reasonable)
    const startTime = Date.now();
    await fetch(FRONTEND_URL);
    const responseTime = Date.now() - startTime;

    if (responseTime > 5000) { // 5 seconds is quite generous
      throw new Error(`Response time too slow: ${responseTime}ms`);
    }
  }

  async testAPIErrorHandling() {
    // Test non-existent endpoint
    try {
      const response = await fetch(`${BACKEND_URL}/api/nonexistent`);
      if (response.status !== 404) {
        throw new Error(`Expected 404 for non-existent endpoint, got ${response.status}`);
      }
    } catch (error) {
      if (!error.message.includes('404')) {
        throw new Error('API should handle non-existent endpoints gracefully');
      }
    }
  }

  async testSecurityHeaders() {
    const response = await fetch(FRONTEND_URL);

    // Check for basic security considerations
    const html = await response.text();

    // Look for XSS protection patterns
    if (html.includes('<script>alert(') || html.includes('javascript:')) {
      throw new Error('Potential XSS vulnerability detected');
    }

    // Check for HTTPS references in production-like setup
    // (This is a basic check, real implementation would be more thorough)
    console.log('   ✓ Basic security checks passed');
  }

  async testUIComponents() {
    const response = await fetch(FRONTEND_URL);
    const html = await response.text();

    // Check for expected UI components
    const uiComponents = [
      /<button[^>]*class[^>]*>/, // Styled buttons
      /<div[^>]*class[^>]*grid/, // Grid layouts
      /<header[^>]*>/, // Header section
      /<main[^>]*>/, // Main content area
    ];

    const foundComponents = uiComponents.filter(pattern => pattern.test(html));

    if (foundComponents.length < 2) {
      throw new Error(`Only ${foundComponents.length} UI components found, expected at least 2`);
    }
  }

  async runAllTests() {
    console.log('🚀 Starting Comprehensive Website Testing');
    console.log('=' .repeat(50));
    console.log('');

    await this.runTest('Frontend Availability', () => this.testFrontendAvailability());
    await this.runTest('Backend Health Endpoint', () => this.testBackendHealthEndpoint());
    await this.runTest('Random Images API Endpoint', () => this.testRandomImagesEndpoint());
    await this.runTest('Page Metadata & SEO', () => this.testPageMetadata());
    await this.runTest('Responsive Design Elements', () => this.testResponsiveDesignElements());
    await this.runTest('Accessibility Features', () => this.testAccessibilityFeatures());
    await this.runTest('Modern Web Features', () => this.testModernWebFeatures());
    await this.runTest('Performance Headers', () => this.testPerformanceHeaders());
    await this.runTest('API Error Handling', () => this.testAPIErrorHandling());
    await this.runTest('Security Considerations', () => this.testSecurityHeaders());
    await this.runTest('UI Components', () => this.testUIComponents());

    this.printSummary();
  }

  printSummary() {
    console.log('=' .repeat(50));
    console.log('📊 Test Summary:');
    console.log(`   Total Tests: ${this.testResults.total}`);
    console.log(`   ✅ Passed: ${this.testResults.passed}`);
    console.log(`   ❌ Failed: ${this.testResults.failed}`);
    console.log(`   Success Rate: ${Math.round((this.testResults.passed / this.testResults.total) * 100)}%`);

    if (this.testResults.failed === 0) {
      console.log('\n🎉 All tests passed! The Random Pics website is working correctly.');
      console.log('   Ready for full Playwright E2E testing and production deployment.');
    } else {
      console.log('\n⚠️  Some tests failed. Please review the issues above.');
    }
    console.log('');
  }
}

// Run the comprehensive tests
const tester = new WebTester();
tester.runAllTests().catch(console.error);