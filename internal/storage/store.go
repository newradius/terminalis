package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Terminalis/internal/models"
)

type Store struct {
	mu       sync.RWMutex
	dataDir  string
	dataFile string
	data     models.SessionData
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dataDir := filepath.Join(home, ".terminalis")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	s := &Store{
		dataDir:  dataDir,
		dataFile: filepath.Join(dataDir, "sessions.json"),
	}

	if err := s.load(); err != nil {
		s.data = models.SessionData{
			Sessions: []models.Session{},
			Folders:  []models.Folder{},
		}
	}

	return s, nil
}

func (s *Store) DataDir() string {
	return s.dataDir
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.dataFile, data, 0600)
}

// Sessions

func (s *Store) GetSessions() []models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Session, len(s.data.Sessions))
	copy(result, s.data.Sessions)
	return result
}

func (s *Store) GetSession(id string) *models.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Sessions {
		if s.data.Sessions[i].ID == id {
			sess := s.data.Sessions[i]
			return &sess
		}
	}
	return nil
}

func (s *Store) SaveSession(sess models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i := range s.data.Sessions {
		if s.data.Sessions[i].ID == sess.ID {
			s.data.Sessions[i] = sess
			found = true
			break
		}
	}
	if !found {
		s.data.Sessions = append(s.data.Sessions, sess)
	}
	return s.save()
}

func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Sessions {
		if s.data.Sessions[i].ID == id {
			s.data.Sessions = append(s.data.Sessions[:i], s.data.Sessions[i+1:]...)
			return s.save()
		}
	}
	return nil
}

// Folders

func (s *Store) GetFolders() []models.Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.Folder, len(s.data.Folders))
	copy(result, s.data.Folders)
	return result
}

func (s *Store) SaveFolder(folder models.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i := range s.data.Folders {
		if s.data.Folders[i].ID == folder.ID {
			s.data.Folders[i] = folder
			found = true
			break
		}
	}
	if !found {
		s.data.Folders = append(s.data.Folders, folder)
	}
	return s.save()
}

func (s *Store) DeleteFolder(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove folder
	for i := range s.data.Folders {
		if s.data.Folders[i].ID == id {
			s.data.Folders = append(s.data.Folders[:i], s.data.Folders[i+1:]...)
			break
		}
	}

	// Move sessions in this folder to root
	for i := range s.data.Sessions {
		if s.data.Sessions[i].FolderID == id {
			s.data.Sessions[i].FolderID = ""
		}
	}

	// Move sub-folders to root
	for i := range s.data.Folders {
		if s.data.Folders[i].ParentID == id {
			s.data.Folders[i].ParentID = ""
		}
	}

	return s.save()
}

func (s *Store) DeleteFolderWithContents(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect all folder IDs to delete (recursive)
	toDelete := map[string]bool{id: true}
	changed := true
	for changed {
		changed = false
		for _, f := range s.data.Folders {
			if toDelete[f.ParentID] && !toDelete[f.ID] {
				toDelete[f.ID] = true
				changed = true
			}
		}
	}

	// Remove sessions in any of the deleted folders
	sessions := make([]models.Session, 0, len(s.data.Sessions))
	for _, sess := range s.data.Sessions {
		if !toDelete[sess.FolderID] {
			sessions = append(sessions, sess)
		}
	}
	s.data.Sessions = sessions

	// Remove all deleted folders
	folders := make([]models.Folder, 0, len(s.data.Folders))
	for _, f := range s.data.Folders {
		if !toDelete[f.ID] {
			folders = append(folders, f)
		}
	}
	s.data.Folders = folders

	return s.save()
}

func (s *Store) GetTree() []models.TreeNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buildTree("")
}

func (s *Store) buildTree(parentID string) []models.TreeNode {
	var nodes []models.TreeNode

	// Add folders at this level
	for _, f := range s.data.Folders {
		if f.ParentID == parentID {
			node := models.TreeNode{
				ID:       f.ID,
				Name:     f.Name,
				Type:     "folder",
				Color:    f.Color,
				Expanded: f.Expanded,
				Children: s.buildTree(f.ID),
			}
			nodes = append(nodes, node)
		}
	}

	// Add sessions at this level
	for _, sess := range s.data.Sessions {
		if sess.FolderID == parentID {
			sessCopy := sess
			node := models.TreeNode{
				ID:      sess.ID,
				Name:    sess.Name,
				Type:    "session",
				Color:   sess.Color,
				Session: &sessCopy,
			}
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// ExportAll returns a copy of all session data for export.
func (s *Store) ExportAll() models.SessionData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := models.SessionData{
		Sessions: make([]models.Session, len(s.data.Sessions)),
		Folders:  make([]models.Folder, len(s.data.Folders)),
	}
	copy(result.Sessions, s.data.Sessions)
	copy(result.Folders, s.data.Folders)
	return result
}

// GetFolderByID returns a pointer to a folder by its ID.
func (s *Store) GetFolderByID(id string) *models.Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Folders {
		if s.data.Folders[i].ID == id {
			f := s.data.Folders[i]
			return &f
		}
	}
	return nil
}

// MoveSession moves a session to a different folder (empty string = root).
func (s *Store) MoveSession(sessionID, newFolderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Sessions {
		if s.data.Sessions[i].ID == sessionID {
			s.data.Sessions[i].FolderID = newFolderID
			return s.save()
		}
	}
	return fmt.Errorf("session not found")
}

// MoveFolder moves a folder to a new parent (empty string = root).
// Returns an error if the move would create a circular reference.
func (s *Store) MoveFolder(folderID, newParentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if folderID == newParentID {
		return fmt.Errorf("cannot move folder into itself")
	}

	// Check for circular reference: newParentID must not be a descendant of folderID
	if newParentID != "" && s.isDescendantOf(newParentID, folderID) {
		return fmt.Errorf("cannot move folder into its own descendant")
	}

	for i := range s.data.Folders {
		if s.data.Folders[i].ID == folderID {
			s.data.Folders[i].ParentID = newParentID
			return s.save()
		}
	}
	return fmt.Errorf("folder not found")
}

// isDescendantOf checks if childID is a descendant of ancestorID.
// Must be called with lock held.
func (s *Store) isDescendantOf(childID, ancestorID string) bool {
	current := childID
	visited := make(map[string]bool)
	for current != "" {
		if current == ancestorID {
			return true
		}
		if visited[current] {
			return false // cycle protection
		}
		visited[current] = true
		found := false
		for _, f := range s.data.Folders {
			if f.ID == current {
				current = f.ParentID
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return false
}

func (s *Store) ToggleFolderExpanded(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Folders {
		if s.data.Folders[i].ID == id {
			s.data.Folders[i].Expanded = !s.data.Folders[i].Expanded
			return s.save()
		}
	}
	return nil
}
