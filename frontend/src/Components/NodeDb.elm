module Components.NodeDb exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.database
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        victoriaMetrics =
                            Point.getText o.node.points Point.typeDbType ""
                                == Point.valueVictoriaMetrics

                        urlPlaceholder =
                            if victoriaMetrics then
                                "http://myserver:8428"

                            else
                                "https://myserver:8086"
                    in
                    [ optionInput Point.typeDbType
                        "Database Type"
                        [ ( Point.valueInfluxDb, "InfluxDB 2.x" )
                        , ( Point.valueVictoriaMetrics, "Victoria Metrics" )
                        ]
                    , textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "URL" urlPlaceholder
                    ]
                        ++ (if victoriaMetrics then
                                []

                            else
                                [ textInput Point.typeOrg "Organization" "org name"
                                , textInput Point.typeBucket "Bucket" "bucket name"
                                ]
                           )
                        ++ [ textInput Point.typeAuthToken "Auth Token" ""
                           , NodeInputs.nodeListInput opts Point.typeTagPointType "Tag Point Types" "Add Point Type"
                           , checkboxInput Point.typeExpandKeyLabels "Expand Key Labels"
                           ]

                else
                    []
               )
