package partition

import (
	"testing"
	"time"
)

func TestPartitioning(t *testing.T) {
	//create a new router table
	table := NewRouterTable("testdb")
	table.ReplicationFactor = 2
	table.Revision = time.Now().Unix()

	table.Entries = append(table.Entries, &RouterEntry{
		Address:    "entry1",
		JsonPort:   8009,
		HttpPort:   8010,
		Partitions: []int{0, 3, 6, 9},
	})

	table.Entries = append(table.Entries, &RouterEntry{
		Address:    "entry3",
		JsonPort:   8009,
		HttpPort:   8010,
		Partitions: []int{1, 4, 7, 10},
	})

	table.Entries = append(table.Entries, &RouterEntry{
		Address:    "entry2",
		JsonPort:   8009,
		HttpPort:   8010,
		Partitions: []int{2, 5, 8, 11},
	})

	//now reload it
	table, err := table.Rebuild()
	if err != nil {
		t.Errorf("Error %s", err)
	}

	//check first partition
	entries, err := table.PartitionEntries(0)
	if err != nil {
		t.Errorf("Error %s", err)
	}

	if len(entries) != 2 {
		t.Errorf("Not enough entries %s", entries)
	}

	if entries[0].Id() != "entry1:8009" {
		t.Errorf("wrong entry[0] %s", entries)
	}

	if entries[1].Id() != "entry3:8009" {
		t.Errorf("wrong entry[1] %s", entries)
	}

	//check partition ring
	entries, err = table.PartitionEntries(11)
	if err != nil {
		t.Errorf("Error %s", err)
	}

	if len(entries) != 2 {
		t.Errorf("Not enough entries %s", entries)
	}

	if entries[0].Id() != "entry2:8009" {
		t.Errorf("wrong entry[0] %s", entries)
	}

	if entries[1].Id() != "entry1:8009" {
		t.Errorf("wrong entry[1] %s", entries)
	}

}
