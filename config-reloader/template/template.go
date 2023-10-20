package template

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	konfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var k8sClient client.Reader

// Render is a go template rendering function it includes all the sprig lib functions
// as well as some extras like a k8sLookup function to get values from k8s objects
// you can access environment variables from the template under .Env
// The passed values will be available under .Values in the templates
func Render(out io.Writer, tmpl string, values interface{}) error {
	t, err := New("tmpl").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(out, map[string]interface{}{
		"Env":    envMap(),
		"Values": values,
	})
}

func Must(t *template.Template, err error) *template.Template {
	if err != nil {
		panic(err)
	}
	return t
}

func New(name string) *template.Template {
	tpl := template.New(name)
	funcMap := sprig.TxtFuncMap()
	funcMap["isLast"] = func(x int, a interface{}) bool {
		return x == reflect.ValueOf(a).Len()-1
	}
	funcMap["include"] = func(name string, data interface{}) (string, error) {
		buf := new(strings.Builder)
		if err := tpl.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	funcMap["tpl"] = func(tmpl string, data interface{}) (string, error) {
		t, err := template.New("").Parse(tmpl)
		if err != nil {
			return "", err
		}
		buf := new(strings.Builder)
		if err := t.Execute(buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	funcMap["toYaml"] = toYaml
	funcMap["fromYaml"] = fromYaml
	funcMap["k8sLookup"] = k8sLookup
	return tpl.Funcs(funcMap).Delims("{{", "}}")
}

func toYaml(v interface{}) (string, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}

func fromYaml(str string) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(str), &m)
	return m, err
}

func k8sLookup(kind, namespace, name string) (map[string]interface{}, error) {
	if k8sClient == nil {
		kfg, err := konfig.GetConfig()
		if err != nil {
			return nil, err
		}
		if c, err := client.New(kfg, client.Options{}); err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		} else {
			k8sClient = c
		}
	}
	gvk, gk := schema.ParseKindArg(kind)
	if gvk == nil {
		// this looks strange but it should make sense if you read the ParseKindArg docs
		gvk = &schema.GroupVersionKind{
			Kind:    gk.Kind,
			Version: gk.Group,
		}
	}
	if name != "" {
		// fetching a single resource by name
		u := unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Kind:    gvk.Kind,
			Version: gvk.Version,
		})
		if err := k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &u); err != nil {
			return nil, fmt.Errorf("failed to get: %w", err)
		}
		return u.UnstructuredContent(), nil
	}
	ul := &unstructured.UnstructuredList{}
	ul.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List", // TODO: is there a better way?
	})
	opts := &client.ListOptions{
		Namespace: namespace,
	}
	if err := k8sClient.List(context.Background(), ul, opts); err != nil {
		return nil, fmt.Errorf("failed to list: %w", err)
	}
	return ul.UnstructuredContent(), nil
}

func envMap() map[string]string {
	envMap := make(map[string]string)

	for _, v := range os.Environ() {
		kv := strings.SplitN(v, "=", 2)
		envMap[kv[0]] = kv[1]
	}
	return envMap
}
