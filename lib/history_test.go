package lib_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naoking158/go-to-trash/lib"
	"github.com/stretchr/testify/assert"
)

// helper: 指定パスのファイルが存在しているかを作成しておく（空ファイル）
func createDummyFile(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(path, []byte("dummy"), 0644)
	assert.NoError(t, err)
}

// helper: 履歴ファイルの各行をパースしてエントリリストを返す
func readHistoryFile(t *testing.T, path string) []lib.ToBeMovedFile {
	t.Helper()
	f, err := os.Open(path)
	assert.NoError(t, err)
	defer f.Close()

	var entries []lib.ToBeMovedFile
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry lib.ToBeMovedFile
		err := json.Unmarshal(scanner.Bytes(), &entry)
		if err == nil {
			entries = append(entries, entry)
		}
	}
	assert.NoError(t, scanner.Err())
	return entries
}

// Test case 1: 履歴ファイルが存在しない場合の新規作成
func TestUpdateHistory_NewFile(t *testing.T) {
	// t.TempDir() で一時ディレクトリを取得
	trashDir := t.TempDir()
	historyPath := filepath.Join(trashDir, lib.HistoryFileName) // historyFileNameは "go-to-trash-history.json"
	// 履歴ファイルはまだ存在しない

	// 新規エントリを2件作成
	entry1 := lib.HistoryEntry{
		From:    "/source/path/file1.txt",
		To:      filepath.Join(trashDir, "trash1.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	entry2 := lib.HistoryEntry{
		From:    "/source/path/file2.txt",
		To:      filepath.Join(trashDir, "trash2.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	// syncHistory 内でチェックするため、エントリの To に対応するダミーファイルを作成
	createDummyFile(t, entry1.To)
	createDummyFile(t, entry2.To)

	hist := lib.NewHistory(historyPath, nil)
	err := hist.UpdateHistory([]lib.HistoryEntry{entry1, entry2})
	assert.NoError(t, err)

	// 履歴ファイルが作成され、2件のエントリが書き込まれていることを確認
	entries := readHistoryFile(t, historyPath)
	assert.Len(t, entries, 2)
	assert.Equal(t, entry1.From, entries[0].From)
	assert.Equal(t, entry2.From, entries[1].From)
}

// Test case 2: 既存の履歴ファイルに対して新規エントリを追記する
func TestUpdateHistory_Append(t *testing.T) {
	trashDir := t.TempDir()
	historyPath := filepath.Join(trashDir, lib.HistoryFileName)

	// 初期エントリを1件作成
	initialEntry := lib.HistoryEntry{
		From:    "/source/path/initial.txt",
		To:      filepath.Join(trashDir, "trash_initial.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	createDummyFile(t, initialEntry.To)

	// 初期履歴ファイルを手動で作成（改行区切り JSON）
	{
		f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		assert.NoError(t, err)
		writer := bufio.NewWriter(f)
		data, err := json.Marshal(initialEntry)
		assert.NoError(t, err)
		_, err = writer.Write(append(data, '\n'))
		assert.NoError(t, err)
		err = writer.Flush()
		assert.NoError(t, err)
		f.Close()
	}

	// LoadHistory を使って既存の履歴を読み込む
	hist, err := lib.LoadHistory(trashDir)
	assert.NoError(t, err)
	// 初期履歴が読み込まれていることを確認
	assert.Len(t, hist.Entries, 1)

	// 新規エントリを作成
	newEntry := lib.HistoryEntry{
		From:    "/source/path/new.txt",
		To:      filepath.Join(trashDir, "trash_new.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	createDummyFile(t, newEntry.To)

	// UpdateHistory で新規エントリを追記
	err = hist.UpdateHistory([]lib.HistoryEntry{newEntry})
	assert.NoError(t, err)

	// 履歴ファイルの中身を確認（初期エントリ + 新規エントリの合計2件）
	entries := readHistoryFile(t, historyPath)
	assert.Len(t, entries, 2)
	assert.Equal(t, initialEntry.From, entries[0].From)
	assert.Equal(t, newEntry.From, entries[1].From)
}

// Test case 3: syncHistory により、存在しないファイルに対応する履歴エントリが削除される
func TestUpdateHistory_Sync(t *testing.T) {
	trashDir := t.TempDir()
	historyPath := filepath.Join(trashDir, lib.HistoryFileName)

	// 有効なエントリと無効なエントリを用意する
	validEntry := lib.HistoryEntry{
		From:    "/source/path/valid.txt",
		To:      filepath.Join(trashDir, "trash_valid.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	invalidEntry := lib.HistoryEntry{
		From:    "/source/path/invalid.txt",
		To:      filepath.Join(trashDir, "trash_invalid.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	// validEntry の To は存在させる。invalidEntry は作成しない
	createDummyFile(t, validEntry.To)

	// 初期履歴ファイルを作成（2件とも書き込み）
	{
		f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		assert.NoError(t, err)
		writer := bufio.NewWriter(f)
		for _, entry := range []lib.HistoryEntry{validEntry, invalidEntry} {
			data, err := json.Marshal(entry)
			assert.NoError(t, err)
			_, err = writer.Write(append(data, '\n'))
			assert.NoError(t, err)
		}
		err = writer.Flush()
		assert.NoError(t, err)
		f.Close()
	}

	// LoadHistory により履歴を読み込む
	hist, err := lib.LoadHistory(trashDir)
	assert.NoError(t, err)
	assert.Len(t, hist.Entries, 2)

	// UpdateHistory に新規エントリは渡さず、syncHistory の効果を確認する
	err = hist.UpdateHistory([]lib.HistoryEntry{})
	assert.NoError(t, err)
	// UpdateHistory 内で syncHistory が実行され、存在しない invalidEntry が history から除かれる
	assert.Len(t, hist.Entries, 1)
	assert.Equal(t, validEntry.From, hist.Entries[0].From)
	// ※なお、履歴ファイル自体は新規エントリがなければ書き換えないため、ディスク上の内容は変わらない点に注意
}

// Test case 4: 新規エントリが空の場合、エラーなく処理が終了する
func TestUpdateHistory_NoNewFiles(t *testing.T) {
	trashDir := t.TempDir()
	historyPath := filepath.Join(trashDir, lib.HistoryFileName)

	// 初期エントリを1件作成
	entry := lib.HistoryEntry{
		From:    "/source/path/entry.txt",
		To:      filepath.Join(trashDir, "trash_entry.txt"),
		Removed: lib.RemovedAt(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
	}
	createDummyFile(t, entry.To)

	// 履歴ファイルを手動作成
	{
		f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		assert.NoError(t, err)
		writer := bufio.NewWriter(f)
		data, err := json.Marshal(entry)
		assert.NoError(t, err)
		_, err = writer.Write(append(data, '\n'))
		assert.NoError(t, err)
		err = writer.Flush()
		assert.NoError(t, err)
		f.Close()
	}

	hist, err := lib.LoadHistory(trashDir)
	assert.NoError(t, err)
	// UpdateHistory に空のスライスを渡す
	err = hist.UpdateHistory([]lib.HistoryEntry{})
	assert.NoError(t, err)

	// 履歴ファイルの内容は変更されず、1件のエントリが保持されているはず
	entries := readHistoryFile(t, historyPath)
	assert.Len(t, entries, 1)
	assert.Equal(t, entry.From, entries[0].From)
}
