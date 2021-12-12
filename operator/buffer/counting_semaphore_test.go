package buffer

// func TestResourceSemaphoreIncrement(t *testing.T) {
// 	testCases := []struct {
// 		desc     string
// 		testFunc func(*testing.T)
// 	}{
// 		{
// 			desc: "Test increment no readers",
// 			testFunc: func(t *testing.T) {
// 				t.Parallel()
// 				rs := NewCountingSemaphore(0)
// 				rs.Increment()
// 				require.Equal(t, int64(1), rs.val)
// 			},
// 		},
// 		{
// 			desc: "Test increment waiting readers",
// 			testFunc: func(t *testing.T) {
// 				t.Parallel()
// 				timeout := 250 * time.Millisecond
// 				rs := NewCountingSemaphore(0)

// 				ctx, cancel := context.WithCancel(context.Background())
// 				defer cancel()

// 				doneChan := make(chan struct{})
// 				go func() {
// 					err := rs.Acquire(ctx)
// 					assert.NoError(t, err)
// 					close(doneChan)
// 				}()

// 				<-time.After(50 * time.Millisecond)

// 				rs.Increment()
// 				require.Equal(t, int64(0), rs.val)
// 				select {
// 				case <-doneChan:
// 				case <-time.After(timeout):
// 					require.Fail(t, "timed out waiting for acquire")
// 				}
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, tc.testFunc)
// 	}
// }

// func TestResourceSemaphoreAcquire(t *testing.T) {
// 	testCases := []struct {
// 		desc     string
// 		testFunc func(*testing.T)
// 	}{
// 		{
// 			desc: "Acquire blocks when 0",
// 			testFunc: func(t *testing.T) {
// 				t.Parallel()
// 				timeout := 250 * time.Millisecond
// 				rs := NewCountingSemaphore(0)

// 				ctx, cancel := context.WithCancel(context.Background())
// 				defer cancel()

// 				doneChan := make(chan struct{})
// 				go func() {
// 					err := rs.Acquire(ctx)
// 					assert.NoError(t, err)
// 					close(doneChan)
// 				}()

// 				<-time.After(50 * time.Millisecond)

// 				select {
// 				case <-doneChan:
// 					require.Fail(t, "Somehow acquired semaphore despite not incrementing")
// 				case <-time.After(timeout):
// 				}
// 			},
// 		},
// 		{
// 			desc: "Acquire works when semaphore val is 1",
// 			testFunc: func(t *testing.T) {
// 				t.Parallel()
// 				timeout := 250 * time.Millisecond
// 				rs := NewCountingSemaphore(1)

// 				ctx, cancel := context.WithCancel(context.Background())
// 				defer cancel()

// 				doneChan := make(chan struct{})
// 				go func() {
// 					err := rs.Acquire(ctx)
// 					assert.NoError(t, err)
// 					close(doneChan)
// 				}()

// 				select {
// 				case <-doneChan:
// 				case <-time.After(timeout):
// 					require.Fail(t, "timed out acquiring semaphore")
// 				}
// 			},
// 		},
// 		{
// 			desc: "Acquire returns when blocked and context cancelled",
// 			testFunc: func(t *testing.T) {
// 				t.Parallel()
// 				timeout := 250 * time.Millisecond
// 				rs := NewCountingSemaphore(0)

// 				ctx, cancel := context.WithCancel(context.Background())

// 				doneChan := make(chan struct{})
// 				go func() {
// 					err := rs.Acquire(ctx)
// 					assert.ErrorIs(t, err, context.Canceled)
// 					close(doneChan)
// 				}()

// 				<-time.After(50 * time.Millisecond)

// 				cancel()

// 				select {
// 				case <-doneChan:
// 				case <-time.After(timeout):
// 					require.Fail(t, "timed out acquiring semaphore")
// 				}
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.desc, tc.testFunc)
// 	}
// }
