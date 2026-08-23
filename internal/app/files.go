package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var errNotReady = errors.New("artifact not ready")

type artifactStore struct {
	root *os.Root
}

func openArtifactStore(path string) (*artifactStore, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &artifactStore{root: root}, nil
}

func (store *artifactStore) Close() error {
	if store == nil || store.root == nil {
		return nil
	}
	return store.root.Close()
}

func (store *artifactStore) mkdirAll(name string, mode os.FileMode) error {
	name, err := cleanLocalPath(name)
	if err != nil {
		return err
	}
	if name == "." {
		return nil
	}
	current := ""
	for _, component := range strings.Split(name, string(filepath.Separator)) {
		if current == "" {
			current = component
		} else {
			current = filepath.Join(current, component)
		}
		info, statErr := store.root.Lstat(current)
		switch {
		case errors.Is(statErr, os.ErrNotExist):
			if mkdirErr := store.root.Mkdir(current, mode); mkdirErr != nil {
				return mkdirErr
			}
		case statErr != nil:
			return statErr
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("refuse symlink path component %q", current)
		case !info.IsDir():
			return fmt.Errorf("path component is not a directory: %q", current)
		}
	}
	return nil
}

func (store *artifactStore) readRegular(name string) ([]byte, error) {
	name, err := cleanLocalPath(name)
	if err != nil {
		return nil, err
	}
	if err := store.validateParents(name); err != nil {
		return nil, err
	}
	info, err := store.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("artifact is not a regular no-follow file: %s", name)
	}
	return store.root.ReadFile(name)
}

func (store *artifactStore) readNonEmpty(name string) ([]byte, error) {
	data, err := store.readRegular(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNotReady
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, errNotReady
	}
	return data, nil
}

func (store *artifactStore) writeAtomic(name string, data []byte, mode os.FileMode) error {
	name, err := cleanLocalPath(name)
	if err != nil {
		return err
	}
	parent := filepath.Dir(name)
	if err := store.mkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := store.validateTarget(name); err != nil {
		return err
	}
	temporary := filepath.Join(parent, ".write-uuter-"+randomToken())
	file, err := store.root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = store.root.Remove(temporary)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := store.root.Rename(temporary, name); err != nil {
		return err
	}
	if err := syncRootDir(store.root, parent); err != nil {
		return err
	}
	keep = true
	return nil
}

func syncRootDir(root *os.Root, name string) error {
	directory, err := root.Open(name)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

// writeAtomicNoReplace commits name only if no entry exists at that name. The
// commit is a single root-relative no-replace rename, so a competitor that
// creates the same name concurrently is never replaced and never observed
// through a check that a later rename could invalidate. It returns the exact
// identity of the committed file, which is what rollback is allowed to remove.
func (store *artifactStore) writeAtomicNoReplace(name string, data []byte, mode os.FileMode) (os.FileInfo, error) {
	name, err := cleanLocalPath(name)
	if err != nil {
		return nil, err
	}
	parent := filepath.Dir(name)
	if err := store.mkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	if err := store.validateParents(name); err != nil {
		return nil, err
	}
	directory, err := store.root.Open(parent)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	parentInfo, err := directory.Stat()
	if err != nil {
		return nil, err
	}
	if !parentInfo.IsDir() {
		return nil, fmt.Errorf("artifact parent is not a directory: %s", parent)
	}
	temporary := ".write-uuter-" + randomToken()
	file, err := store.root.OpenFile(filepath.Join(parent, temporary), os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = store.root.Remove(filepath.Join(parent, temporary))
		}
	}()
	if _, err := file.Write(data); err != nil {
		return nil, err
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	// The rename preserves this identity, so it names the committed file even
	// if the directory entry is later taken over by something else.
	committed, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	if err := renameNoReplaceIn(directory, temporary, filepath.Base(name)); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("target already exists: %s", name)
		}
		return nil, err
	}
	if err := directory.Sync(); err != nil {
		return nil, err
	}
	keep = true
	return committed, nil
}

// renameNoReplaceIn performs the platform no-replace rename relative to an
// already-open directory handle, so no path component is resolved again.
func renameNoReplaceIn(directory *os.File, oldName, newName string) error {
	connection, err := directory.SyscallConn()
	if err != nil {
		return err
	}
	var renameErr error
	if controlErr := connection.Control(func(descriptor uintptr) {
		renameErr = renameNoReplaceAt(descriptor, oldName, newName)
	}); controlErr != nil {
		return controlErr
	}
	return renameErr
}

// renameNoReplacePath commits a no-replace rename between two paths that share
// a parent directory, binding that parent as an open handle first.
func renameNoReplacePath(oldPath, newPath string) error {
	parent := filepath.Dir(newPath)
	if filepath.Dir(oldPath) != parent {
		return fmt.Errorf("no-replace rename requires a shared parent directory")
	}
	directory, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := renameNoReplaceIn(directory, filepath.Base(oldPath), filepath.Base(newPath)); err != nil {
		return err
	}
	return directory.Sync()
}

// removeOwned deletes name only while it is still the exact file identity this
// controller committed. A competing file that took the name is left alone.
func (store *artifactStore) removeOwned(name string, owned os.FileInfo) error {
	if owned == nil {
		return nil
	}
	name, err := cleanLocalPath(name)
	if err != nil {
		return err
	}
	if err := store.validateParents(name); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	current, err := store.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(owned, current) {
		return fmt.Errorf("refuse to remove a competing artifact: %s", name)
	}
	err = store.root.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (store *artifactStore) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return store.writeAtomic(name, append(data, '\n'), 0o644)
}

func (store *artifactStore) remove(name string) error {
	name, err := cleanLocalPath(name)
	if err != nil {
		return err
	}
	if err := store.validateParents(name); err != nil {
		return err
	}
	err = store.root.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (store *artifactStore) removeAll(name string) error {
	name, err := cleanLocalPath(name)
	if err != nil {
		return err
	}
	if err := store.validateParents(name); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return store.root.RemoveAll(name)
}

func (store *artifactStore) copyRegularFrom(source *artifactStore, sourceName, targetName string, mode os.FileMode) ([]byte, error) {
	data, err := source.readRegular(sourceName)
	if err != nil {
		return nil, err
	}
	if err := store.writeAtomic(targetName, data, mode); err != nil {
		return nil, err
	}
	return data, nil
}

func (store *artifactStore) copyTreeFrom(source *artifactStore, sourceName, targetName string) error {
	sourceName, err := cleanLocalPath(sourceName)
	if err != nil {
		return err
	}
	if err := source.validateParents(sourceName); err != nil {
		return err
	}
	rootInfo, err := source.root.Lstat(sourceName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("refuse non-directory or symlink tree root: %s", sourceName)
	}
	entries, err := fs.ReadDir(source.root.FS(), sourceName)
	if err != nil {
		return err
	}
	if err := store.mkdirAll(targetName, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		sourceChild := filepath.Join(sourceName, entry.Name())
		targetChild := filepath.Join(targetName, entry.Name())
		info, infoErr := source.root.Lstat(sourceChild)
		if infoErr != nil {
			return infoErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse symlink artifact: %s", sourceChild)
		}
		if info.IsDir() {
			if err := store.copyTreeFrom(source, sourceChild, targetChild); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refuse non-regular artifact: %s", sourceChild)
		}
		if _, err := store.copyRegularFrom(source, sourceChild, targetChild, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func (store *artifactStore) validateParents(name string) error {
	parent := filepath.Dir(name)
	if parent == "." {
		return nil
	}
	current := ""
	for _, component := range strings.Split(parent, string(filepath.Separator)) {
		if current == "" {
			current = component
		} else {
			current = filepath.Join(current, component)
		}
		info, err := store.root.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe artifact path component: %s", current)
		}
	}
	return nil
}

func (store *artifactStore) validateTarget(name string) error {
	if err := store.validateParents(name); err != nil {
		return err
	}
	info, err := store.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to replace non-regular artifact: %s", name)
	}
	return nil
}

func cleanLocalPath(name string) (string, error) {
	name = filepath.Clean(name)
	if name == "." {
		return name, nil
	}
	if !filepath.IsLocal(name) {
		return "", fmt.Errorf("artifact path is not local: %q", name)
	}
	return name, nil
}

func randomToken() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(bytes[:])
}

func revisionFor(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func reviewDigest(result, report []byte) string {
	combined := make([]byte, 0, len(result)+len(report)+1)
	combined = append(combined, result...)
	combined = append(combined, 0)
	combined = append(combined, report...)
	return revisionFor(combined)
}
