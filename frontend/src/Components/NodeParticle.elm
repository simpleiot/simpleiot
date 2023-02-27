module Components.NodeParticle exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Dict exposing (Dict)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Round
import Time
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.Iso8601 as Iso8601


view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisable ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.particle
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts ""

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""
                    in
                    [ text "Particle.io connection"
                    , textInput Point.typeDescription "Description" ""
                    , textInput Point.typeAuthToken "API Key" ""
                    , checkboxInput Point.typeDisable "Disable"
                    , viewPoints o.zone <| Point.filterSpecialPoints <| List.sortWith Point.sort o.node.points
                    ]

                else
                    []
               )


viewPoints : Time.Zone -> List Point.Point -> Element msg
viewPoints z pts =
    let
        formaters =
            metricFormaters z

        fm =
            formatMetric formaters
    in
    table [ padding 7 ]
        { data = List.map (fm z) pts
        , columns =
            let
                cell =
                    el [ paddingXY 15 5, Border.width 1 ]
            in
            [ { header = cell <| el [ Font.bold, centerX ] <| text "Time"
              , width = fill
              , view = \m -> cell <| text m.time
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "ID"
              , width = fill
              , view = \m -> cell <| text m.key
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Type"
              , width = fill
              , view = \m -> cell <| text m.desc
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
              , width = fill
              , view = \m -> cell <| el [ alignRight ] <| text m.value
              }
            ]
        }


formatMetric : Dict String MetricFormat -> Time.Zone -> Point.Point -> { time : String, key : String, desc : String, value : String }
formatMetric formaters z p =
    case Dict.get p.typ formaters of
        Just f ->
            { time = Iso8601.toString Iso8601.Second z p.time
            , key = p.key
            , desc = f.desc p
            , value = f.vf p
            }

        Nothing ->
            { time = Iso8601.toString Iso8601.Second z p.time
            , key = p.key
            , desc = p.typ
            , value = toOneDec p
            }


type alias MetricFormat =
    { desc : Point.Point -> String
    , vf : Point.Point -> String
    }


metricFormaters : Time.Zone -> Dict String MetricFormat
metricFormaters _ =
    Dict.fromList
        [ ( "temp", { desc = descS "Temperature", vf = toOneDec } )
        ]


descS : String -> Point.Point -> String
descS d _ =
    d


toOneDec : Point.Point -> String
toOneDec p =
    Round.round 1 p.value
