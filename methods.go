package main

import "sync/atomic"

// NextIndex atomically increase the counter and return an index
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextRRPeer() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next

	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}

	return nil
}

// GetNextWRRPeer returns next active weighted peer to take a connection
func (s *ServerPool) GetNextWRRPeer() *Backend {
	if len(s.backends) == 0 {
		return nil
	}

	if s.weight == 0 {
		s.current = (s.current + 1) % uint64(len(s.backends))
		s.weight = s.backends[s.current].Weight
	}

	server := s.backends[s.current]
	s.weight--
	return server
}
