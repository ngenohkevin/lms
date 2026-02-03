package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngenohkevin/lms/internal/middleware"
	"github.com/ngenohkevin/lms/internal/models"
	"github.com/ngenohkevin/lms/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AllowAllPermissionService always returns true for HasPermission (for testing)
type AllowAllPermissionService struct{}

func (s *AllowAllPermissionService) ListPermissions(ctx context.Context) (*models.PermissionsListResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) GetPermissionByCode(ctx context.Context, code string) (*models.Permission, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) GetPermissionMatrix(ctx context.Context) (*models.PermissionMatrixResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) GetRolePermissions(ctx context.Context, role models.UserRole) (*models.RolePermissionsResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) UpdateRolePermissions(ctx context.Context, role models.UserRole, permissionCodes []string, grantedByUserID int) error {
	return nil
}
func (s *AllowAllPermissionService) GetRolePermissionCodes(ctx context.Context, role models.UserRole) ([]string, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) GetUserEffectivePermissions(ctx context.Context, userID int, username string, role models.UserRole) (*models.UserEffectivePermissionsResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) GetMyPermissions(ctx context.Context, userID int, role models.UserRole) (*models.MyPermissionsResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	return true, nil // Always allow for tests
}
func (s *AllowAllPermissionService) HasAnyPermission(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	return true, nil
}
func (s *AllowAllPermissionService) HasAllPermissions(ctx context.Context, userID int, permissionCodes []string) (bool, error) {
	return true, nil
}
func (s *AllowAllPermissionService) ListUserOverrides(ctx context.Context, userID int, username string) (*models.UserOverridesResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) CreateUserOverride(ctx context.Context, userID int, req *models.CreateUserOverrideRequest, grantedByUserID int) (*models.UserOverrideResponse, error) {
	return nil, nil
}
func (s *AllowAllPermissionService) DeleteUserOverride(ctx context.Context, userID int, permissionCode string) error {
	return nil
}
func (s *AllowAllPermissionService) InvalidateUserCache(ctx context.Context, userID int) error {
	return nil
}
func (s *AllowAllPermissionService) InvalidateRoleCache(ctx context.Context, role models.UserRole) error {
	return nil
}

func createTestPermissionMiddleware() *middleware.PermissionMiddleware {
	return middleware.NewPermissionMiddleware(&AllowAllPermissionService{})
}

// MockFineService is a mock implementation of FineService
type MockFineService struct {
	mock.Mock
}

func (m *MockFineService) ListFines(ctx context.Context, paid *bool, studentID *int32, page, limit int32) (*services.FineListResult, error) {
	args := m.Called(ctx, paid, studentID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.FineListResult), args.Error(1)
}

func (m *MockFineService) GetFine(ctx context.Context, transactionID int32) (*services.Fine, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Fine), args.Error(1)
}

func (m *MockFineService) GetUnpaidFinesByStudent(ctx context.Context, studentID int32) ([]services.UnpaidFine, error) {
	args := m.Called(ctx, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]services.UnpaidFine), args.Error(1)
}

func (m *MockFineService) GetTotalUnpaidFines(ctx context.Context, studentID int32) (float64, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockFineService) PayFine(ctx context.Context, transactionID int32) (*services.Fine, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Fine), args.Error(1)
}

func (m *MockFineService) WaiveFine(ctx context.Context, transactionID int32, waivedBy int32, reason string) (*services.Fine, error) {
	args := m.Called(ctx, transactionID, waivedBy, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Fine), args.Error(1)
}

func (m *MockFineService) GetFineStatistics(ctx context.Context) (*services.FineStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.FineStatistics), args.Error(1)
}

func (m *MockFineService) CalculateFinesForOverdueBooks(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockFineService) GetStudentsWithHighFines(ctx context.Context, threshold float64) ([]services.StudentWithHighFines, error) {
	args := m.Called(ctx, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]services.StudentWithHighFines), args.Error(1)
}

func (m *MockFineService) GetFinePerDay() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

func (m *MockFineService) BulkPayFines(ctx context.Context, transactionIDs []int32) (int64, error) {
	args := m.Called(ctx, transactionIDs)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockFineService) BulkWaiveFines(ctx context.Context, transactionIDs []int32, waivedBy int32, reason string) (int64, error) {
	args := m.Called(ctx, transactionIDs, waivedBy, reason)
	return args.Get(0).(int64), args.Error(1)
}

// DenyAllPermissionService denies all permissions (for testing forbidden scenarios)
type DenyAllPermissionService struct {
	AllowAllPermissionService
}

func (s *DenyAllPermissionService) HasPermission(ctx context.Context, userID int, permissionCode string) (bool, error) {
	return false, nil // Always deny for tests
}

func createDenyPermissionMiddleware() *middleware.PermissionMiddleware {
	return middleware.NewPermissionMiddleware(&DenyAllPermissionService{})
}

func setupFineTestRouter(handler *FineHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add middleware to set user ID (required for permission check)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", 1)
		c.Next()
	})
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, createTestPermissionMiddleware())
	return router
}

// setupFineTestRouterNoAuth sets up a router WITHOUT auth context (for testing unauthorized)
func setupFineTestRouterNoAuth(handler *FineHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// NO user_id set - simulating unauthenticated request
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, createTestPermissionMiddleware())
	return router
}

// setupFineTestRouterDenyPerm sets up a router with permission denied (for testing forbidden)
func setupFineTestRouterDenyPerm(handler *FineHandler, userID int32, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add middleware to set auth context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int(userID))
		c.Set("user_role", role)
		c.Next()
	})
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, createDenyPermissionMiddleware()) // Use deny middleware
	return router
}

// setupFineTestRouterWithAuth sets up a router with mock auth context for privileged operations
func setupFineTestRouterWithAuth(handler *FineHandler, userID int32, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Add middleware to set auth context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int(userID))
		c.Set("user_role", role)
		c.Next()
	})
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api, createTestPermissionMiddleware())
	return router
}

func TestFineHandler_ListFines(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully lists fines", func(t *testing.T) {
		result := &services.FineListResult{
			Fines: []services.Fine{
				{
					TransactionID: 1,
					StudentID:     1,
					StudentName:   "John Doe",
					Amount:        5.0,
					Paid:          false,
				},
			},
			Total:      1,
			Page:       1,
			Limit:      20,
			TotalPages: 1,
		}

		mockService.On("ListFines", mock.Anything, (*bool)(nil), (*int32)(nil), int32(1), int32(20)).Return(result, nil).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("returns error on service failure", func(t *testing.T) {
		mockService.On("ListFines", mock.Anything, (*bool)(nil), (*int32)(nil), int32(1), int32(20)).Return(nil, errors.New("db error")).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestFineHandler_GetFine(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully gets fine by ID", func(t *testing.T) {
		fine := &services.Fine{
			TransactionID: 1,
			StudentID:     1,
			StudentName:   "John Doe",
			Amount:        5.0,
			Paid:          false,
		}

		mockService.On("GetFine", mock.Anything, int32(1)).Return(fine, nil).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/fines/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns error when fine not found", func(t *testing.T) {
		mockService.On("GetFine", mock.Anything, int32(999)).Return(nil, errors.New("not found")).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestFineHandler_GetUnpaidFinesByStudent(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully gets unpaid fines for student", func(t *testing.T) {
		fines := []services.UnpaidFine{
			{
				TransactionID: 1,
				BookID:        1,
				BookTitle:     "Test Book",
				Amount:        5.0,
			},
		}

		mockService.On("GetUnpaidFinesByStudent", mock.Anything, int32(1)).Return(fines, nil).Once()
		mockService.On("GetTotalUnpaidFines", mock.Anything, int32(1)).Return(5.0, nil).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/student/1/unpaid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})
}

func TestFineHandler_PayFine(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully pays fine", func(t *testing.T) {
		fine := &services.Fine{
			TransactionID: 1,
			StudentID:     1,
			StudentName:   "John Doe",
			Amount:        5.0,
			Paid:          true,
		}

		mockService.On("PayFine", mock.Anything, int32(1)).Return(fine, nil).Once()

		req, _ := http.NewRequest("POST", "/api/v1/fines/1/pay", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("returns error on pay failure", func(t *testing.T) {
		mockService.On("PayFine", mock.Anything, int32(2)).Return(nil, errors.New("fine not found")).Once()

		req, _ := http.NewRequest("POST", "/api/v1/fines/2/pay", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestFineHandler_WaiveFine(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	// Use auth router with admin role for waive operations
	router := setupFineTestRouterWithAuth(handler, 1, "admin")

	t.Run("successfully waives fine", func(t *testing.T) {
		reason := "Student hardship"
		fine := &services.Fine{
			TransactionID: 1,
			StudentID:     1,
			StudentName:   "John Doe",
			Amount:        5.0,
			Waived:        true,
			WaivedReason:  &reason,
		}

		mockService.On("WaiveFine", mock.Anything, int32(1), int32(1), "Student hardship").Return(fine, nil).Once()

		body := WaiveFineRequest{Reason: "Student hardship"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/fines/1/waive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("returns error when reason is missing", func(t *testing.T) {
		body := WaiveFineRequest{}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/fines/1/waive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns unauthorized when user_id not in context", func(t *testing.T) {
		// Use router without auth context
		noAuthRouter := setupFineTestRouterNoAuth(handler)

		body := WaiveFineRequest{Reason: "Test reason"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/fines/1/waive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		noAuthRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns forbidden when user does not have permission", func(t *testing.T) {
		// Use router with deny permission middleware (simulates denied permission override)
		denyRouter := setupFineTestRouterDenyPerm(handler, 1, "student")

		body := WaiveFineRequest{Reason: "Test reason"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/fines/1/waive", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		denyRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestFineHandler_GetFineStatistics(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully gets fine statistics", func(t *testing.T) {
		stats := &services.FineStatistics{
			UnpaidCount:             10,
			PaidCount:               50,
			WaivedCount:             5,
			TotalUnpaid:             100.0,
			TotalCollected:          500.0,
			TotalWaived:             25.0,
			StudentsWithUnpaidFines: 8,
		}

		mockService.On("GetFineStatistics", mock.Anything).Return(stats, nil).Once()
		mockService.On("GetFinePerDay").Return(0.50).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/statistics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})
}

func TestFineHandler_CalculateFines(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully calculates fines", func(t *testing.T) {
		mockService.On("CalculateFinesForOverdueBooks", mock.Anything).Return(5, nil).Once()

		req, _ := http.NewRequest("POST", "/api/v1/fines/calculate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("returns error on calculation failure", func(t *testing.T) {
		mockService.On("CalculateFinesForOverdueBooks", mock.Anything).Return(0, errors.New("db error")).Once()

		req, _ := http.NewRequest("POST", "/api/v1/fines/calculate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestFineHandler_GetStudentsWithHighFines(t *testing.T) {
	mockService := new(MockFineService)
	handler := NewFineHandler(mockService)
	router := setupFineTestRouter(handler)

	t.Run("successfully gets students with high fines", func(t *testing.T) {
		students := []services.StudentWithHighFines{
			{
				StudentID:   1,
				StudentCode: "STU001",
				StudentName: "John Doe",
				TotalFines:  25.0,
				FineCount:   5,
			},
		}

		mockService.On("GetStudentsWithHighFines", mock.Anything, 10.0).Return(students, nil).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/high-fines?threshold=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response.Success)
	})

	t.Run("uses default threshold when not provided", func(t *testing.T) {
		students := []services.StudentWithHighFines{}

		mockService.On("GetStudentsWithHighFines", mock.Anything, 10.0).Return(students, nil).Once()

		req, _ := http.NewRequest("GET", "/api/v1/fines/high-fines", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
