package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/entity"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/storage"
)

const ticketFileName = "ticket.md"
const conclusionFileName = "conclusion.md"

type Store struct {
	*entity.BaseStore
	locks sync.Map
}

func (s *Store) ticketMu(id string) *sync.Mutex {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func NewStore(ticketsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	base, err := entity.NewBaseStore(ticketsDir, bus, projectPath)
	if err != nil {
		return nil, err
	}

	s := &Store{BaseStore: base}

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		dir := filepath.Join(ticketsDir, string(status))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return s, nil
}

func (s *Store) Create(title, body string, dueDate *time.Time, references []string, repo string) (*Ticket, error) {
	if title == "" {
		return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
	}

	now := time.Now().UTC()

	ticketID, err := storage.NewTicketIDFromCreated(now, title, s.RootDir())
	if err != nil {
		return nil, fmt.Errorf("generate ticket ID: %w", err)
	}

	ticket := &Ticket{
		ID:     ticketID,
		Status: StatusBacklog,
		TicketMeta: TicketMeta{
			Title:      title,
			Repo:       repo,
			References: references,
			Due:        dueDate,
			Created:    now,
			Updated:    now,
		},
		Body: body,
	}

	mu := s.ticketMu(ticket.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.saveTicket(ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketCreated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) Get(id string) (*Ticket, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		entityDir := filepath.Join(s.RootDir(), string(status), id)
		if info, statErr := os.Stat(entityDir); statErr != nil || !info.IsDir() {
			continue
		}
		ticket, err := s.loadFromDir(entityDir)
		if err != nil {
			return nil, "", err
		}
		ticket.ID = id
		ticket.Status = status
		return ticket, status, nil
	}

	return nil, "", &NotFoundError{Resource: "ticket", ID: id}
}

func (s *Store) Update(id string, title, body *string, references *[]string) (*Ticket, error) {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, status, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadFromDir(entityDir)
	if err != nil {
		return nil, err
	}
	ticket.ID = id
	ticket.Status = status

	titleChanged := false
	if title != nil {
		if *title == "" {
			return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
		}
		if ticket.Title != *title {
			titleChanged = true
		}
		ticket.Title = *title
	}
	if body != nil {
		ticket.Body = *body
	}
	if references != nil {
		ticket.References = *references
	}

	ticket.Updated = time.Now().UTC()

	if titleChanged {
		newID, err := storage.NewTicketIDFromCreated(ticket.Created, ticket.Title, s.RootDir())
		if err != nil {
			return nil, fmt.Errorf("generate new ticket ID: %w", err)
		}
		newDir := filepath.Join(s.RootDir(), string(status), newID)
		if err := os.Rename(entityDir, newDir); err != nil {
			return nil, fmt.Errorf("rename entity dir: %w", err)
		}
		entityDir = newDir
		id = newID
		ticket.ID = newID
	}

	if err := s.writeFile(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) EditBody(id, oldString, newString string, replaceAll bool) (*Ticket, error) {
	if oldString == "" {
		return nil, &ValidationError{Field: "oldString", Message: "cannot be empty"}
	}
	if oldString == newString {
		return nil, &ValidationError{Field: "newString", Message: "must differ from oldString"}
	}

	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, status, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadFromDir(entityDir)
	if err != nil {
		return nil, err
	}
	ticket.ID = id
	ticket.Status = status

	matches := findBodyEditMatches(ticket.Body, oldString)
	if len(matches) == 0 {
		return nil, &ValidationError{Field: "oldString", Message: "could not find the target text in the ticket body"}
	}
	if len(matches) > 1 && !replaceAll {
		return nil, &ValidationError{Field: "oldString", Message: "matched multiple locations; use replaceAll=true or provide more surrounding context"}
	}

	ticket.Body = applyBodyEditMatches(ticket.Body, matches, newString, replaceAll)
	ticket.Updated = time.Now().UTC()

	if err := s.writeFile(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

type bodyEditMatch struct {
	start int
	end   int
}

func findBodyEditMatches(content, oldString string) []bodyEditMatch {
	for _, matcher := range []func(string, string) []bodyEditMatch{
		findExactBodyEditMatches,
		findNormalizedBodyEditMatches,
		findAnchoredBodyEditMatches,
	} {
		if matches := matcher(content, oldString); len(matches) > 0 {
			return matches
		}
	}
	return nil
}

func findExactBodyEditMatches(content, oldString string) []bodyEditMatch {
	var matches []bodyEditMatch
	for offset := 0; ; {
		idx := strings.Index(content[offset:], oldString)
		if idx == -1 {
			return matches
		}
		start := offset + idx
		matches = append(matches, bodyEditMatch{start: start, end: start + len(oldString)})
		offset = start + len(oldString)
	}
}

func findNormalizedBodyEditMatches(content, oldString string) []bodyEditMatch {
	contentLines, contentStarts := splitLinesWithOffsets(content)
	searchLines := normalizeSearchLines(oldString)
	if len(searchLines) == 0 || len(searchLines) > len(contentLines) {
		return nil
	}

	var matches []bodyEditMatch
	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		matched := true
		for j := range searchLines {
			if normalizeBodyEditLine(contentLines[i+j]) != normalizeBodyEditLine(searchLines[j]) {
				matched = false
				break
			}
		}
		if matched {
			start := contentStarts[i]
			lastLine := i + len(searchLines) - 1
			end := contentStarts[lastLine] + len(contentLines[lastLine])
			matches = append(matches, bodyEditMatch{start: start, end: end})
		}
	}

	return matches
}

func findAnchoredBodyEditMatches(content, oldString string) []bodyEditMatch {
	contentLines, contentStarts := splitLinesWithOffsets(content)
	searchLines := normalizeSearchLines(oldString)
	if len(searchLines) < 3 {
		return nil
	}

	firstIdx, lastIdx, ok := firstAndLastNonEmptyLine(searchLines)
	if !ok || firstIdx == lastIdx {
		return nil
	}

	searchSequence := collectNormalizedNonEmptyLines(searchLines[firstIdx : lastIdx+1])
	if len(searchSequence) < 2 {
		return nil
	}

	firstAnchor := normalizeBodyEditLine(searchLines[firstIdx])
	lastAnchor := normalizeBodyEditLine(searchLines[lastIdx])

	var matches []bodyEditMatch
	for startLine := 0; startLine < len(contentLines); startLine++ {
		if normalizeBodyEditLine(contentLines[startLine]) != firstAnchor {
			continue
		}
		for endLine := startLine + 1; endLine < len(contentLines); endLine++ {
			if normalizeBodyEditLine(contentLines[endLine]) != lastAnchor {
				continue
			}
			candidateSequence := collectNormalizedNonEmptyLines(contentLines[startLine : endLine+1])
			if !equalStrings(candidateSequence, searchSequence) {
				continue
			}

			start := contentStarts[startLine]
			end := contentStarts[endLine] + len(contentLines[endLine])
			matches = append(matches, bodyEditMatch{start: start, end: end})
			break
		}
	}

	return matches
}

func applyBodyEditMatches(content string, matches []bodyEditMatch, newString string, replaceAll bool) string {
	if !replaceAll {
		match := matches[0]
		return content[:match.start] + newString + content[match.end:]
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		b.WriteString(content[last:match.start])
		b.WriteString(newString)
		last = match.end
	}
	b.WriteString(content[last:])
	return b.String()
}

func splitLinesWithOffsets(content string) ([]string, []int) {
	lines := strings.Split(content, "\n")
	starts := make([]int, len(lines))
	offset := 0
	for i, line := range lines {
		starts[i] = offset
		offset += len(line)
		if i < len(lines)-1 {
			offset++
		}
	}
	return lines, starts
}

func normalizeSearchLines(search string) []string {
	lines := strings.Split(search, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func firstAndLastNonEmptyLine(lines []string) (int, int, bool) {
	first := -1
	last := -1
	for i, line := range lines {
		if normalizeBodyEditLine(line) == "" {
			continue
		}
		if first == -1 {
			first = i
		}
		last = i
	}
	if first == -1 || last == -1 {
		return 0, 0, false
	}
	return first, last, true
}

func collectNormalizedNonEmptyLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		normalized := normalizeBodyEditLine(line)
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func normalizeBodyEditLine(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *Store) SetDueDate(id string, dueDate *time.Time) (*Ticket, error) {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, status, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadFromDir(entityDir)
	if err != nil {
		return nil, err
	}
	ticket.ID = id
	ticket.Status = status

	ticket.Due = dueDate
	ticket.Updated = time.Now().UTC()

	if err := s.writeFile(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) ClearDueDate(id string) (*Ticket, error) {
	return s.SetDueDate(id, nil)
}

func (s *Store) Delete(id string) error {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(entityDir); err != nil {
		return fmt.Errorf("remove entity directory: %w", err)
	}

	s.locks.Delete(id)
	s.Emit(events.TicketDeleted, id, nil)
	return nil
}

func (s *Store) List(status Status) ([]*Ticket, error) {
	dir := filepath.Join(s.RootDir(), string(status))
	entityDirs, err := s.ListEntries(dir)
	if err != nil {
		return nil, err
	}

	var tickets []*Ticket
	for _, entityDir := range entityDirs {
		ticket, err := s.loadFromDir(entityDir)
		if err != nil {
			return nil, err
		}
		ticket.ID = filepath.Base(entityDir)
		ticket.Status = status
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func (s *Store) ListAll() (map[Status][]*Ticket, error) {
	result := make(map[Status][]*Ticket)

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		tickets, err := s.List(status)
		if err != nil {
			return nil, err
		}
		result[status] = tickets
	}

	return result, nil
}

func (s *Store) Move(id string, to Status) error {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, from, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return err
	}

	if from == to {
		return nil
	}

	ticket, err := s.loadFromDir(entityDir)
	if err != nil {
		return err
	}
	ticket.ID = id
	ticket.Status = from

	ticket.Updated = time.Now().UTC()

	toDir := filepath.Join(s.RootDir(), string(to))
	dirName := filepath.Base(entityDir)
	newDir := filepath.Join(toDir, dirName)
	if err := os.Rename(entityDir, newDir); err != nil {
		return fmt.Errorf("move entity dir: %w", err)
	}

	if err := s.writeFile(newDir, ticket); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketMoved, ticket.ID, nil)
	return nil
}

func (s *Store) FilePath(id string) (string, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(entityDir, ticketFileName), nil
}

func (s *Store) HasConclusion(id string) (bool, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return false, err
	}
	_, statErr := os.Stat(filepath.Join(entityDir, conclusionFileName))
	return statErr == nil, nil
}

func (s *Store) WriteConclusion(ticketID string, meta *TicketConclusionMeta, body string) error {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(ticketID)
	if err != nil {
		return fmt.Errorf("find ticket dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(meta, body)
	if err != nil {
		return fmt.Errorf("serialize conclusion: %w", err)
	}

	return storage.AtomicWriteFile(filepath.Join(entityDir, conclusionFileName), data)
}

func (s *Store) ReadConclusion(ticketID string) (*TicketConclusionMeta, string, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(ticketID)
	if err != nil {
		return nil, "", fmt.Errorf("find ticket dir: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(entityDir, conclusionFileName))
	if err != nil {
		return nil, "", err
	}

	return storage.ParseFrontmatter[TicketConclusionMeta](data)
}

type TicketConclusionMeta struct {
	StartedAt       time.Time `yaml:"started_at"`
	ConcludedAt     time.Time `yaml:"concluded_at"`
	Agent           string    `yaml:"agent"`
	Profile         string    `yaml:"profile,omitempty"`
	Rejected        bool      `yaml:"rejected,omitempty"`
	RejectionReason string    `yaml:"rejection_reason,omitempty"`
	Commits         []string  `yaml:"commits,omitempty"`
}

func (s *Store) saveTicket(ticket *Ticket) error {
	entityDir := filepath.Join(s.RootDir(), string(ticket.Status), ticket.ID)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("create entity dir: %w", err)
	}

	return s.writeFile(entityDir, ticket)
}

func (s *Store) writeFile(entityDir string, ticket *Ticket) error {
	data, err := storage.SerializeFrontmatter(&ticket.TicketMeta, ticket.Body)
	if err != nil {
		return fmt.Errorf("serialize ticket: %w", err)
	}

	return s.WriteFileBytes(entityDir, ticketFileName, data)
}

func (s *Store) loadFromDir(entityDir string) (*Ticket, error) {
	data, err := s.LoadFileBytes(entityDir, ticketFileName)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ticketFileName, err)
	}

	keys, err := storage.FrontmatterKeys(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s keys: %w", ticketFileName, err)
	}
	if _, ok := keys["path"]; ok {
		return nil, fmt.Errorf("parse %s: legacy frontmatter field %q is no longer supported; migrate this ticket before using Cortex", ticketFileName, "path")
	}

	meta, body, err := storage.ParseFrontmatter[TicketMeta](data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", ticketFileName, err)
	}

	return &Ticket{
		TicketMeta: *meta,
		Body:       body,
	}, nil
}

func (s *Store) findEntityDir(id string) (string, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		target := filepath.Join(s.RootDir(), string(status), id)
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return target, nil
		}
	}
	return "", &NotFoundError{Resource: "ticket", ID: id}
}

func (s *Store) findEntityDirAllStatuses(id string) (string, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		dir := filepath.Join(s.RootDir(), string(status), id)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, status, nil
		}
	}
	return "", "", &NotFoundError{Resource: "ticket", ID: id}
}
