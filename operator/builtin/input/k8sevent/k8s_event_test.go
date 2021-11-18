package k8sevent

import (
	"context"
	"testing"
	"time"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	fakev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	fakeTest "k8s.io/client-go/testing"
)

var fakeTime = time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)

type fakeWatch struct{}

func (f *fakeWatch) Stop() {}
func (f *fakeWatch) ResultChan() <-chan watch.Event {
	ch := make(chan watch.Event, 1)
	ch <- watch.Event{
		Type: "ADDED",
		Object: (&apiv1.Event{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
			InvolvedObject: apiv1.ObjectReference{
				Kind:      "Pod",
				Name:      "testpodname",
				UID:       types.UID("testuid"),
				Namespace: "testnamespace",
			},
			LastTimestamp: metav1.Time{
				Time: fakeTime,
			},
		}).DeepCopyObject(),
	}
	return ch
}

func TestWatchNamespace(t *testing.T) {
	inputOp, err := helper.NewInputConfig("test_id", "k8s_event_input").Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	fakeAPI := &fakeTest.Fake{}
	fakeAPI.AddWatchReactor("*", func(action fakeTest.Action) (handled bool, ret watch.Interface, err error) {
		return true, &fakeWatch{}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	op := &K8sEvents{
		InputOperator: inputOp,
		client: &fakev1.FakeCoreV1{
			Fake: fakeAPI,
		},
		namespaces: []string{"test_namespace"},
		cancel:     cancel,
	}

	fake := testutil.NewFakeOutput(t)
	op.OutputOperators = []operator.Operator{fake}

	op.startWatchingNamespace(ctx, "test_namespace")
	defer op.Stop()

	select {
	case entry := <-fake.Received:
		require.Equal(t, entry.Timestamp, fakeTime)
		require.Equal(t, entry.Resource["k8s.namespace.name"], "testnamespace")
		require.Equal(t, entry.Resource["k8s.pod.uid"], "testuid")
		require.Equal(t, entry.Resource["k8s.pod.name"], "testpodname")
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry")
	}
}

func TestListNamespaces(t *testing.T) {
	fakeAPI := &fakeTest.Fake{}
	fakeAPI.AddReactor("*", "*", func(action fakeTest.Action) (bool, runtime.Object, error) {
		list := apiv1.NamespaceList{
			Items: []apiv1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test2",
					},
				},
			},
		}
		return true, list.DeepCopyObject(), nil
	})
	fakeClient := &fakev1.FakeCoreV1{
		Fake: fakeAPI,
	}

	namespaces, err := listNamespaces(context.Background(), fakeClient)
	require.NoError(t, err)
	require.Equal(t, []string{"test1", "test2"}, namespaces)
}
