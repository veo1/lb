package main

import "sync/atomic"

// NextIndex atomically increase the counter and return an index
func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextRRPeer() *Backend {
	// loop entire backends to find out an Alive backend
	next := s.NextIndex()
	l := len(s.backends) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(s.backends)     // take an index by modding
		if s.backends[idx].IsAlive() { // if we have an alive backend, use it and store if its not the original one
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

// GetNextPeer returns next active peer to take a connection
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
