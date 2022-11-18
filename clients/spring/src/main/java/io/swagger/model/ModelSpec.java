package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobShardingConfig;
import io.swagger.model.ModelJobSpecDocker;
import io.swagger.model.ModelJobSpecLanguage;
import io.swagger.model.ModelJobSpecWasm;
import io.swagger.model.ModelResourceUsageConfig;
import io.swagger.model.ModelStorageSpec;
import io.swagger.v3.oas.annotations.media.Schema;
import java.math.BigDecimal;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * ModelSpec
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class ModelSpec   {
  @JsonProperty("Annotations")
  @Valid
  private List<String> annotations = null;

  @JsonProperty("Contexts")
  @Valid
  private List<ModelStorageSpec> contexts = null;

  @JsonProperty("DoNotTrack")
  private Boolean doNotTrack = null;

  @JsonProperty("Docker")
  private ModelJobSpecDocker docker = null;

  @JsonProperty("Engine")
  private Integer engine = null;

  @JsonProperty("Language")
  private ModelJobSpecLanguage language = null;

  @JsonProperty("Publisher")
  private Integer publisher = null;

  @JsonProperty("Resources")
  private ModelResourceUsageConfig resources = null;

  @JsonProperty("Sharding")
  private ModelJobShardingConfig sharding = null;

  @JsonProperty("Timeout")
  private BigDecimal timeout = null;

  @JsonProperty("Verifier")
  private Integer verifier = null;

  @JsonProperty("Wasm")
  private ModelJobSpecWasm wasm = null;

  @JsonProperty("inputs")
  @Valid
  private List<ModelStorageSpec> inputs = null;

  @JsonProperty("outputs")
  @Valid
  private List<ModelStorageSpec> outputs = null;

  public ModelSpec annotations(List<String> annotations) {
    this.annotations = annotations;
    return this;
  }

  public ModelSpec addAnnotationsItem(String annotationsItem) {
    if (this.annotations == null) {
      this.annotations = new ArrayList<String>();
    }
    this.annotations.add(annotationsItem);
    return this;
  }

  /**
   * Annotations on the job - could be user or machine assigned
   * @return annotations
   **/
  @Schema(description = "Annotations on the job - could be user or machine assigned")
  
    public List<String> getAnnotations() {
    return annotations;
  }

  public void setAnnotations(List<String> annotations) {
    this.annotations = annotations;
  }

  public ModelSpec contexts(List<ModelStorageSpec> contexts) {
    this.contexts = contexts;
    return this;
  }

  public ModelSpec addContextsItem(ModelStorageSpec contextsItem) {
    if (this.contexts == null) {
      this.contexts = new ArrayList<ModelStorageSpec>();
    }
    this.contexts.add(contextsItem);
    return this;
  }

  /**
   * Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes
   * @return contexts
   **/
  @Schema(description = "Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes")
      @Valid
    public List<ModelStorageSpec> getContexts() {
    return contexts;
  }

  public void setContexts(List<ModelStorageSpec> contexts) {
    this.contexts = contexts;
  }

  public ModelSpec doNotTrack(Boolean doNotTrack) {
    this.doNotTrack = doNotTrack;
    return this;
  }

  /**
   * Do not track specified by the client
   * @return doNotTrack
   **/
  @Schema(description = "Do not track specified by the client")
  
    public Boolean isDoNotTrack() {
    return doNotTrack;
  }

  public void setDoNotTrack(Boolean doNotTrack) {
    this.doNotTrack = doNotTrack;
  }

  public ModelSpec docker(ModelJobSpecDocker docker) {
    this.docker = docker;
    return this;
  }

  /**
   * Get docker
   * @return docker
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobSpecDocker getDocker() {
    return docker;
  }

  public void setDocker(ModelJobSpecDocker docker) {
    this.docker = docker;
  }

  public ModelSpec engine(Integer engine) {
    this.engine = engine;
    return this;
  }

  /**
   * e.g. docker or language
   * @return engine
   **/
  @Schema(description = "e.g. docker or language")
  
    public Integer getEngine() {
    return engine;
  }

  public void setEngine(Integer engine) {
    this.engine = engine;
  }

  public ModelSpec language(ModelJobSpecLanguage language) {
    this.language = language;
    return this;
  }

  /**
   * Get language
   * @return language
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobSpecLanguage getLanguage() {
    return language;
  }

  public void setLanguage(ModelJobSpecLanguage language) {
    this.language = language;
  }

  public ModelSpec publisher(Integer publisher) {
    this.publisher = publisher;
    return this;
  }

  /**
   * there can be multiple publishers for the job
   * @return publisher
   **/
  @Schema(description = "there can be multiple publishers for the job")
  
    public Integer getPublisher() {
    return publisher;
  }

  public void setPublisher(Integer publisher) {
    this.publisher = publisher;
  }

  public ModelSpec resources(ModelResourceUsageConfig resources) {
    this.resources = resources;
    return this;
  }

  /**
   * Get resources
   * @return resources
   **/
  @Schema(description = "")
  
    @Valid
    public ModelResourceUsageConfig getResources() {
    return resources;
  }

  public void setResources(ModelResourceUsageConfig resources) {
    this.resources = resources;
  }

  public ModelSpec sharding(ModelJobShardingConfig sharding) {
    this.sharding = sharding;
    return this;
  }

  /**
   * Get sharding
   * @return sharding
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobShardingConfig getSharding() {
    return sharding;
  }

  public void setSharding(ModelJobShardingConfig sharding) {
    this.sharding = sharding;
  }

  public ModelSpec timeout(BigDecimal timeout) {
    this.timeout = timeout;
    return this;
  }

  /**
   * How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results
   * @return timeout
   **/
  @Schema(description = "How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results")
  
    @Valid
    public BigDecimal getTimeout() {
    return timeout;
  }

  public void setTimeout(BigDecimal timeout) {
    this.timeout = timeout;
  }

  public ModelSpec verifier(Integer verifier) {
    this.verifier = verifier;
    return this;
  }

  /**
   * Get verifier
   * @return verifier
   **/
  @Schema(description = "")
  
    public Integer getVerifier() {
    return verifier;
  }

  public void setVerifier(Integer verifier) {
    this.verifier = verifier;
  }

  public ModelSpec wasm(ModelJobSpecWasm wasm) {
    this.wasm = wasm;
    return this;
  }

  /**
   * Get wasm
   * @return wasm
   **/
  @Schema(description = "")
  
    @Valid
    public ModelJobSpecWasm getWasm() {
    return wasm;
  }

  public void setWasm(ModelJobSpecWasm wasm) {
    this.wasm = wasm;
  }

  public ModelSpec inputs(List<ModelStorageSpec> inputs) {
    this.inputs = inputs;
    return this;
  }

  public ModelSpec addInputsItem(ModelStorageSpec inputsItem) {
    if (this.inputs == null) {
      this.inputs = new ArrayList<ModelStorageSpec>();
    }
    this.inputs.add(inputsItem);
    return this;
  }

  /**
   * the data volumes we will read in the job for example \"read this ipfs cid\" TODO: #667 Replace with \"Inputs\", \"Outputs\" (note the caps) for yaml/json when we update the n.js file
   * @return inputs
   **/
  @Schema(description = "the data volumes we will read in the job for example \"read this ipfs cid\" TODO: #667 Replace with \"Inputs\", \"Outputs\" (note the caps) for yaml/json when we update the n.js file")
      @Valid
    public List<ModelStorageSpec> getInputs() {
    return inputs;
  }

  public void setInputs(List<ModelStorageSpec> inputs) {
    this.inputs = inputs;
  }

  public ModelSpec outputs(List<ModelStorageSpec> outputs) {
    this.outputs = outputs;
    return this;
  }

  public ModelSpec addOutputsItem(ModelStorageSpec outputsItem) {
    if (this.outputs == null) {
      this.outputs = new ArrayList<ModelStorageSpec>();
    }
    this.outputs.add(outputsItem);
    return this;
  }

  /**
   * the data volumes we will write in the job for example \"write the results to ipfs\"
   * @return outputs
   **/
  @Schema(description = "the data volumes we will write in the job for example \"write the results to ipfs\"")
      @Valid
    public List<ModelStorageSpec> getOutputs() {
    return outputs;
  }

  public void setOutputs(List<ModelStorageSpec> outputs) {
    this.outputs = outputs;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    ModelSpec modelSpec = (ModelSpec) o;
    return Objects.equals(this.annotations, modelSpec.annotations) &&
        Objects.equals(this.contexts, modelSpec.contexts) &&
        Objects.equals(this.doNotTrack, modelSpec.doNotTrack) &&
        Objects.equals(this.docker, modelSpec.docker) &&
        Objects.equals(this.engine, modelSpec.engine) &&
        Objects.equals(this.language, modelSpec.language) &&
        Objects.equals(this.publisher, modelSpec.publisher) &&
        Objects.equals(this.resources, modelSpec.resources) &&
        Objects.equals(this.sharding, modelSpec.sharding) &&
        Objects.equals(this.timeout, modelSpec.timeout) &&
        Objects.equals(this.verifier, modelSpec.verifier) &&
        Objects.equals(this.wasm, modelSpec.wasm) &&
        Objects.equals(this.inputs, modelSpec.inputs) &&
        Objects.equals(this.outputs, modelSpec.outputs);
  }

  @Override
  public int hashCode() {
    return Objects.hash(annotations, contexts, doNotTrack, docker, engine, language, publisher, resources, sharding, timeout, verifier, wasm, inputs, outputs);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class ModelSpec {\n");
    
    sb.append("    annotations: ").append(toIndentedString(annotations)).append("\n");
    sb.append("    contexts: ").append(toIndentedString(contexts)).append("\n");
    sb.append("    doNotTrack: ").append(toIndentedString(doNotTrack)).append("\n");
    sb.append("    docker: ").append(toIndentedString(docker)).append("\n");
    sb.append("    engine: ").append(toIndentedString(engine)).append("\n");
    sb.append("    language: ").append(toIndentedString(language)).append("\n");
    sb.append("    publisher: ").append(toIndentedString(publisher)).append("\n");
    sb.append("    resources: ").append(toIndentedString(resources)).append("\n");
    sb.append("    sharding: ").append(toIndentedString(sharding)).append("\n");
    sb.append("    timeout: ").append(toIndentedString(timeout)).append("\n");
    sb.append("    verifier: ").append(toIndentedString(verifier)).append("\n");
    sb.append("    wasm: ").append(toIndentedString(wasm)).append("\n");
    sb.append("    inputs: ").append(toIndentedString(inputs)).append("\n");
    sb.append("    outputs: ").append(toIndentedString(outputs)).append("\n");
    sb.append("}");
    return sb.toString();
  }

  /**
   * Convert the given object to string with each line indented by 4 spaces
   * (except the first line).
   */
  private String toIndentedString(java.lang.Object o) {
    if (o == null) {
      return "null";
    }
    return o.toString().replace("\n", "\n    ");
  }
}
