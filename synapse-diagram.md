::: mermaid
graph LR
linkedService.AzureDataLakeStorage1 --> integrationRuntime.AutoResolveIntegrationRuntime
linkedService.gitsynapsetesting2it-WorkspaceDefaultSqlServer --> integrationRuntime.AutoResolveIntegrationRuntime
linkedService.gitsynapsetesting2it-WorkspaceDefaultStorage --> integrationRuntime.AutoResolveIntegrationRuntime
pipeline.pipeline1 --> notebook.Notebook2
dataset.dataset1 --> linkedService.gitsynapsetesting2it-WorkspaceDefaultStorage
sparkJobDefinition.Spark_job_definition_1 --> bigDataPool.pool2
sparkJobDefinition.SparkDefinition1 --> bigDataPool.pool1
sparkJobDefinition.SparkDefinition2 --> bigDataPool.pool1
:::
