import axios from 'axios';

// Create an axios instance with a custom config
const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  }
});

// Request interceptor
apiClient.interceptors.request.use(
  config => {
    // You could add authentication headers here if needed
    return config;
  },
  error => {
    return Promise.reject(error);
  }
);

// Response interceptor
apiClient.interceptors.response.use(
  response => {
    // Unwrap the data from the API response format
    const apiResponse = response.data;
    
    if (apiResponse.code === 0) {
      // Success - return just the data portion
      return apiResponse.data;
    } else {
      // API returned an error code
      return Promise.reject({
        message: apiResponse.message || 'Unknown error',
        code: apiResponse.code
      });
    }
  },
  error => {
    // Handle HTTP errors
    let message = 'Network error';
    if (error.response) {
      message = `Server error: ${error.response.status}`;
    } else if (error.request) {
      message = 'No response from server';
    }
    
    return Promise.reject({
      message: message,
      originalError: error
    });
  }
);

export default apiClient;