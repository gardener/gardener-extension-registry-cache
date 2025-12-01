<p>Packages:</p>
<ul>
<li>
<a href="#registry.extensions.gardener.cloud%2fv1alpha3">registry.extensions.gardener.cloud/v1alpha3</a>
</li>
</ul>
<h2 id="registry.extensions.gardener.cloud/v1alpha3">registry.extensions.gardener.cloud/v1alpha3</h2>
<p>
<p>Package v1alpha3 is a version of the API.</p>
</p>
Resource Types:
<ul></ul>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.GarbageCollection">GarbageCollection
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache</a>)
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
<code>ttl</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.34/#duration-v1-meta">
Kubernetes meta/v1.Duration
</a>
</em>
</td>
<td>
<p>TTL is the time to live of a blob in the cache.
Set to 0s to disable the garbage collection.
Defaults to 168h (7 days).</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.HTTP">HTTP
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache</a>)
</p>
<p>
<p>HTTP contains settings for the HTTP server that hosts the registry cache.</p>
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
<code>tls</code></br>
<em>
bool
</em>
</td>
<td>
<p>TLS indicates whether TLS is enabled for the HTTP server of the registry cache.
Defaults to true.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.HighAvailability">HighAvailability
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache</a>)
</p>
<p>
<p>HighAvailability contains settings for high availability of the registry cache.</p>
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
<p>Enabled defines if the registry cache is scaled with the high availability feature.
For more details, see <a href="https://github.com/gardener/gardener/blob/master/docs/development/high-availability-of-components.md#system-components">https://github.com/gardener/gardener/blob/master/docs/development/high-availability-of-components.md#system-components</a>.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.Proxy">Proxy
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache</a>)
</p>
<p>
<p>Proxy contains settings for a proxy used in the registry cache.</p>
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
<code>httpProxy</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTPProxy field represents the proxy server for HTTP connections which is used by the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>httpsProxy</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTPSProxy field represents the proxy server for HTTPS connections which is used by the registry cache.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryConfig">RegistryConfig</a>)
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
The value must be a valid DNS subdomain (RFC 1123) and optionally a port.</p>
</td>
</tr>
<tr>
<td>
<code>remoteURL</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RemoteURL is the remote registry URL. The format must be <code>&lt;scheme&gt;&lt;host&gt;[:&lt;port&gt;]</code> where
<code>&lt;scheme&gt;</code> is <code>https://</code> or <code>http://</code> and <code>&lt;host&gt;[:&lt;port&gt;]</code> corresponds to the Upstream</p>
<p>If defined, the value is set as <code>proxy.remoteurl</code> in the registry <a href="https://github.com/distribution/distribution/blob/main/docs/content/recipes/mirror.md#configure-the-cache">configuration</a>
and in containerd configuration as <code>server</code> field in <a href="https://github.com/containerd/containerd/blob/main/docs/hosts.md#server-field">hosts.toml</a> file.</p>
</td>
</tr>
<tr>
<td>
<code>volume</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.Volume">
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
<a href="#registry.extensions.gardener.cloud/v1alpha3.GarbageCollection">
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
<tr>
<td>
<code>proxy</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.Proxy">
Proxy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Proxy contains settings for a proxy used in the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>http</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.HTTP">
HTTP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTP contains settings for the HTTP server that hosts the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>highAvailability</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.HighAvailability">
HighAvailability
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HighAvailability contains settings for high availability of the registry cache.</p>
</td>
</tr>
<tr>
<td>
<code>serviceNameSuffix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceNameSuffix allows to customize the naming of the deployed service.
If not specified, the service suffix will be generated from the upstream.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.RegistryCacheStatus">RegistryCacheStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryStatus">RegistryStatus</a>)
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
<p>Upstream is the remote registry host (and optionally port).</p>
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
Examples: &ldquo;<a href="https://10.4.246.205:5000&quot;">https://10.4.246.205:5000&rdquo;</a>, &ldquo;<a href="http://10.4.26.127:5000&quot;">http://10.4.26.127:5000&rdquo;</a></p>
</td>
</tr>
<tr>
<td>
<code>remoteURL</code></br>
<em>
string
</em>
</td>
<td>
<p>RemoteURL is the remote registry URL.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="registry.extensions.gardener.cloud/v1alpha3.RegistryConfig">RegistryConfig
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
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">
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
<h3 id="registry.extensions.gardener.cloud/v1alpha3.RegistryStatus">RegistryStatus
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
<code>caSecretName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CASecretName is the name of the CA bundle secret.
The field is nil when there is no registry cache that enables TLS for the HTTP server.</p>
</td>
</tr>
<tr>
<td>
<code>caches</code></br>
<em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCacheStatus">
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
<h3 id="registry.extensions.gardener.cloud/v1alpha3.Volume">Volume
</h3>
<p>
(<em>Appears on:</em>
<a href="#registry.extensions.gardener.cloud/v1alpha3.RegistryCache">RegistryCache</a>)
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
<a href="https://godoc.org/k8s.io/apimachinery/pkg/api/resource#Quantity">
k8s.io/apimachinery/pkg/api/resource.Quantity
</a>
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
