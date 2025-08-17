package sessions

import (
	"sync"
	"time"

	"textsurf/modules"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/google/uuid"
)

// Manager 会话管理器
type Manager struct {
	sessions map[string]*modules.Session
	mutex    sync.RWMutex
}

// NewManager 创建新的会话管理器
func NewManager() *Manager {
	manager := &Manager{
		sessions: make(map[string]*modules.Session),
	}

	// 启动清理过期会话的 goroutine
	go manager.cleanupExpiredSessions()

	return manager
}

// CreateSession 创建新会话
func (m *Manager) CreateSession(module modules.Module, headless bool) (*modules.Session, error) {
	// 启动浏览器
	url := launcher.New().
		Headless(headless).
		MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()

	// 创建会话
	session := &modules.Session{
		ID:        uuid.New().String(),
		Browser:   browser,
		CreatedAt: time.Now(),
		Module:    module,
		Data:      make(map[string]interface{}),
	}

	// 存储会话
	m.mutex.Lock()
	m.sessions[session.ID] = session
	m.mutex.Unlock()

	return session, nil
}

// GetSession 获取会话
func (m *Manager) GetSession(sessionID string) (*modules.Session, bool) {
	m.mutex.RLock()
	session, exists := m.sessions[sessionID]
	m.mutex.RUnlock()
	return session, exists
}

// DeleteSession 删除会话
func (m *Manager) DeleteSession(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if session, exists := m.sessions[sessionID]; exists {
		// 关闭浏览器资源
		session.Module.Close(session)
		delete(m.sessions, sessionID)
	}
}

// cleanupExpiredSessions 清理过期会话 (超过1小时)
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			if now.Sub(session.CreatedAt) > time.Hour {
				session.Module.Close(session)
				delete(m.sessions, id)
			}
		}
		m.mutex.Unlock()
	}
}

// ListSessions 获取所有会话ID
func (m *Manager) ListSessions() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}
