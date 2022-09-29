<p>Packages:</p>
<ul>
<li>
<a href="#registry.extensions.gardener.cloud%2fv1alpha1">registry.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="registry.extensions.gardener.cloud/v1alpha1">registry.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the registry service extension.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="registry.extensions.gardener.cloud/v1alpha1.RegistryCache">RegistryCache
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha1.RegistryConfig">RegistryConfig</a>)
</p>
<p>
<p>RegistryCache defines a registry cache to deploy</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>upstream</code></br>
<em>
string
</em>
</td>
<td>
<p>Upstream is the remote registry host (and optionally port) to cache</p>
</td>
</tr>
<tr>
<td>
<code>size</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<em>(Optional)</em>
<p>Size is the size of the registry cache, defaults to 10Gi.</p>
</td>
</tr>
<tr>
<td>
<code>garbageCollectionEnabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>GarbageCollectionEnabled enables/disables cache garbage collection, defaults to true.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha1.RegistryConfig">RegistryConfig
</h3>
<p>
<p>RegistryConfig configuration resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>caches</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha1.RegistryCache">
[]RegistryCache
</a>
</em>
</td>
<td>
<p>Caches is a slice of registry cache to deploy</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
