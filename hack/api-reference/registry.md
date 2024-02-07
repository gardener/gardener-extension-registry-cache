<p>Packages:</p>
<ul>
<li>
<a href="#registry.extensions.gardener.cloud%2fv1alpha2">registry.extensions.gardener.cloud/v1alpha2</a>
</li>
</ul>
<h2 id="registry.extensions.gardener.cloud/v1alpha2">registry.extensions.gardener.cloud/v1alpha2</h2>
<p>
<p>Package v1alpha2 is a version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.GarbageCollection">GarbageCollection
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryCache">RegistryCache</a>)
</p>
<p>
<p>GarbageCollection contains settings for the garbage collection of content from the cache.</p>
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
<code>enabled</code></br>
<em>
bool
</em>
</td>
<td>
<p>Enabled indicates whether the garbage collection is enabled.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.RegistryCache">RegistryCache
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryConfig">RegistryConfig</a>)
</p>
<p>
<p>RegistryCache represents a registry cache to deploy.</p>
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
<p>Upstream is the remote registry host to cache.
The value must be a valid DNS subdomain (RFC 1123).</p>
</td>
</tr>
<tr>
<td>
<code>volume</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.Volume">
Volume
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Volume contains settings for the registry cache volume.</p>
</td>
</tr>
<tr>
<td>
<code>garbageCollection</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.GarbageCollection">
GarbageCollection
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>GarbageCollection contains settings for the garbage collection of content from the cache.
Defaults to enabled garbage collection.</p>
</td>
</tr>
<tr>
<td>
<code>secretReferenceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SecretReferenceName is the name of the reference for the Secret containing the upstream registry credentials.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.RegistryCacheStatus">RegistryCacheStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryStatus">RegistryStatus</a>)
</p>
<p>
<p>RegistryCacheStatus represents a deployed registry cache.</p>
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
<p>Upstream is the remote registry host.</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>Endpoint is the registry cache endpoint.
Example: &ldquo;<a href="http://10.4.246.205:5000&quot;">http://10.4.246.205:5000&rdquo;</a></p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.RegistryConfig">RegistryConfig
</h3>
<p>
<p>RegistryConfig contains information about registry caches to deploy.</p>
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
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryCache">
[]RegistryCache
</a>
</em>
</td>
<td>
<p>Caches is a slice of registry caches to deploy.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.RegistryStatus">RegistryStatus
</h3>
<p>
<p>RegistryStatus contains information about deployed registry caches.</p>
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
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryCacheStatus">
[]RegistryCacheStatus
</a>
</em>
</td>
<td>
<p>Caches is a slice of deployed registry caches.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha2.Volume">Volume
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha2.RegistryCache">RegistryCache</a>)
</p>
<p>
<p>Volume contains settings for the registry cache volume.</p>
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
<code>size</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<em>(Optional)</em>
<p>Size is the size of the registry cache volume.
Defaults to 10Gi.
This field is immutable.</p>
</td>
</tr>
<tr>
<td>
<code>storageClassName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageClassName is the name of the StorageClass used by the registry cache volume.
This field is immutable.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
