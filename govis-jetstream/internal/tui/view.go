package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString("STREAMS  (↑/↓: select, r: refresh, q: quit)\n\n")

	switch {
	case m.streamsLoading:
		b.WriteString("Loading streams...")
	case m.streamsErr != nil:
		fmt.Fprintf(&b, "Error: %v", m.streamsErr)
	case len(m.streams) == 0:
		b.WriteString("No streams found")
	default:
		fmt.Fprintf(&b, "  %-20s %12s %10s %10s %13s\n", "NAME", "STORED MSGS", "MAX MSGS", "CONSUMERS", "MAX CONSUMERS")
		for i, stream := range m.streams {
			marker := " "
			if i == m.selected {
				marker = ">"
			}

			fmt.Fprintf(
				&b, "%s %-20s %12d %10s %10d %13s\n",
				marker,
				stream.Name,
				stream.NumMessages,
				formatLimit(stream.MaxMessages),
				stream.NumConsumers,
				formatLimit(int64(stream.MaxConsumers)),
			)
		}
		if m.selected >= 0 && m.selected < len(m.streams) {
			selected := m.streams[m.selected]
			if selected.NumMessages == 0 {
				fmt.Fprintf(&b, "\n%s: no retained messages\n", selected.Name)
			} else {
				fmt.Fprintf(&b, "\n%s retained sequences: %d–%d (oldest–newest)\n", selected.Name, selected.FirstSeq, selected.LastSeq)
			}
		}
		fmt.Fprintf(&b, "\nLIVE FLOW: %s (observed publishes / aggregate delivery attempts)\n", m.selectedStream())
		fmt.Fprintf(&b, "  %-10s %s STREAM  %.1f/s\n", "PUBLISHER", flowLane(m.publishParticles, m.animationNow), m.currentPublishRate())
		if m.newEventSubject != "" {
			fmt.Fprintf(&b, "  Last observed: %s", m.newEventSubject)
			if m.newEventJobID != "" {
				fmt.Fprintf(&b, "  ID: %s", m.newEventJobID)
			}
			b.WriteByte('\n')
		}
		if m.consumerSnapshotsErr != nil {
			fmt.Fprintf(&b, "  Consumer state: %v\n", m.consumerSnapshotsErr)
		}
		switch {
		case m.consumersLoading && len(m.consumers) == 0:
			b.WriteString("  Loading consumers...\n")
		case m.consumersErr != nil && len(m.consumers) == 0:
			fmt.Fprintf(&b, "  Error: %v\n", m.consumersErr)
		case len(m.consumers) == 0:
			b.WriteString("  No consumers\n")
		default:
			for _, consumer := range m.consumers {
				fmt.Fprintf(&b, "  %-10s %s %s", "STREAM", flowLane(m.consumerParticles[consumer], m.animationNow), consumer)
				var snapshotFound bool
				for _, snapshot := range m.consumerSnapshots {
					if snapshot.Consumer == consumer {
						fmt.Fprintf(&b, "  %.1f/s  waiting:%d  in-flight:%d", m.consumerRates[consumer], snapshot.NumPending, snapshot.NumAckPending)
						snapshotFound = true
						break
					}
				}
				if !snapshotFound {
					b.WriteString("  waiting for snapshot")
				}
				b.WriteByte('\n')
			}
		}

		fmt.Fprintf(&b, "\nSUBJECTS IN: %s (currently stored)\n", m.selectedStream())
		switch {
		case m.subjectsLoading && len(m.subjects) == 0:
			b.WriteString("Loading...")
		case m.subjectsErr != nil && len(m.subjects) == 0:
			fmt.Fprintf(&b, "Error: %v", m.subjectsErr)
		case len(m.subjects) == 0:
			b.WriteString("No subjects with stored messages")
		default:
			for _, subject := range m.subjects {
				fmt.Fprintf(&b, "  %s\n", subject)
			}
		}

	}

	return tea.NewView(b.String())
}

func formatLimit(limit int64) string {
	if limit == -1 {
		return "unlimited"
	}
	return strconv.FormatInt(limit, 10)
}
