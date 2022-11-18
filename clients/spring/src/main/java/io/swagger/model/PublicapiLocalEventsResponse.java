package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobLocalEvent;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiLocalEventsResponse
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiLocalEventsResponse   {
  @JsonProperty("localEvents")
  @Valid
  private List<ModelJobLocalEvent> localEvents = null;

  public PublicapiLocalEventsResponse localEvents(List<ModelJobLocalEvent> localEvents) {
    this.localEvents = localEvents;
    return this;
  }

  public PublicapiLocalEventsResponse addLocalEventsItem(ModelJobLocalEvent localEventsItem) {
    if (this.localEvents == null) {
      this.localEvents = new ArrayList<ModelJobLocalEvent>();
    }
    this.localEvents.add(localEventsItem);
    return this;
  }

  /**
   * Get localEvents
   * @return localEvents
   **/
  @Schema(description = "")
      @Valid
    public List<ModelJobLocalEvent> getLocalEvents() {
    return localEvents;
  }

  public void setLocalEvents(List<ModelJobLocalEvent> localEvents) {
    this.localEvents = localEvents;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiLocalEventsResponse publicapiLocalEventsResponse = (PublicapiLocalEventsResponse) o;
    return Objects.equals(this.localEvents, publicapiLocalEventsResponse.localEvents);
  }

  @Override
  public int hashCode() {
    return Objects.hash(localEvents);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiLocalEventsResponse {\n");
    
    sb.append("    localEvents: ").append(toIndentedString(localEvents)).append("\n");
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
