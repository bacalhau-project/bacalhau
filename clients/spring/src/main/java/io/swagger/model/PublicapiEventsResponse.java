package io.swagger.model;

import java.util.Objects;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonCreator;
import io.swagger.model.ModelJobEvent;
import io.swagger.v3.oas.annotations.media.Schema;
import java.util.ArrayList;
import java.util.List;
import org.springframework.validation.annotation.Validated;
import javax.validation.Valid;
import javax.validation.constraints.*;

/**
 * PublicapiEventsResponse
 */
@Validated
@javax.annotation.Generated(value = "io.swagger.codegen.v3.generators.java.SpringCodegen", date = "2022-11-25T18:06:37.098869Z[Europe/London]")


public class PublicapiEventsResponse   {
  @JsonProperty("events")
  @Valid
  private List<ModelJobEvent> events = null;

  public PublicapiEventsResponse events(List<ModelJobEvent> events) {
    this.events = events;
    return this;
  }

  public PublicapiEventsResponse addEventsItem(ModelJobEvent eventsItem) {
    if (this.events == null) {
      this.events = new ArrayList<ModelJobEvent>();
    }
    this.events.add(eventsItem);
    return this;
  }

  /**
   * Get events
   * @return events
   **/
  @Schema(description = "")
      @Valid
    public List<ModelJobEvent> getEvents() {
    return events;
  }

  public void setEvents(List<ModelJobEvent> events) {
    this.events = events;
  }


  @Override
  public boolean equals(java.lang.Object o) {
    if (this == o) {
      return true;
    }
    if (o == null || getClass() != o.getClass()) {
      return false;
    }
    PublicapiEventsResponse publicapiEventsResponse = (PublicapiEventsResponse) o;
    return Objects.equals(this.events, publicapiEventsResponse.events);
  }

  @Override
  public int hashCode() {
    return Objects.hash(events);
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append("class PublicapiEventsResponse {\n");
    
    sb.append("    events: ").append(toIndentedString(events)).append("\n");
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
