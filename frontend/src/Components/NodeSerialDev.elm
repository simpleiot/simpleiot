module Components.NodeSerialDev exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Dict exposing (Dict)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import FormatNumber exposing (format)
import FormatNumber.Locales exposing (Decimals(..), usLocale)
import Round
import Time
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.Iso8601 exposing (toDateTimeString)


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
            [ Icon.serialDev
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            180

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts ""

                        numberInput =
                            NodeInputs.nodeNumberInput opts ""

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts ""

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""

                        log =
                            Point.getText o.node.points Point.typeLog ""

                        rate =
                            Point.getValue o.node.points Point.typeRate ""

                        rateS =
                            String.fromFloat (Round.roundNum 0 rate)
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typePort "Port" "/dev/ttyUSB0"
                    , textInput Point.typeBaud "Baud" "9600"
                    , numberInput Point.typeMaxMessageLength "Max Msg Len"
                    , numberInput Point.typeDebug "Debug level (0-9)"
                    , checkboxInput Point.typeDisable "Disable"
                    , counterWithReset Point.typeErrorCount Point.typeErrorCountReset "Error Count"
                    , counterWithReset Point.typeRx Point.typeRxReset "Rx count"
                    , counterWithReset Point.typeTx Point.typeTxReset "Tx count"
                    , text <| "  Rate (pts/sec): " ++ rateS
                    , text <| "  Last log: " ++ log
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
        { data = List.map fm pts
        , columns =
            let
                cell =
                    el [ paddingXY 15 5, Border.width 1 ]
            in
            [ { header = cell <| el [ Font.bold, centerX ] <| text "Point"
              , width = fill
              , view = \m -> cell <| text m.desc
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
              , width = fill
              , view = \m -> cell <| el [ alignRight ] <| text m.value
              }
            ]
        }


formatMetric : Dict String MetricFormat -> Point.Point -> { desc : String, value : String }
formatMetric formaters p =
    case Dict.get p.typ formaters of
        Just f ->
            { desc = f.desc p, value = f.vf p }

        Nothing ->
            Point.renderPoint2 p


type alias MetricFormat =
    { desc : Point.Point -> String
    , vf : Point.Point -> String
    }


metricFormaters : Time.Zone -> Dict String MetricFormat
metricFormaters z =
    let
        toTimeWithZone =
            toTime z
    in
    Dict.fromList
        [ ( "metricAppAlloc", { desc = descS "App Memory Alloc", vf = toMiB } )
        , ( "metricAppNumGoroutine", { desc = descS "App Goroutine Count", vf = toWhole } )
        , ( "metricProcCPUPercent", { desc = descS "Proc CPU %", vf = toPercent } )
        , ( "metricProcMemPercent", { desc = descS "Proc Mem %", vf = toPercent } )
        , ( "metricProcMemRSS", { desc = descS "Proc Mem RSS", vf = toMiB } )
        , ( "host", { desc = descKey "Host", vf = toText } )
        , ( "hostBootTime", { desc = descS "Host Boot Time", vf = toTimeWithZone } )
        , ( "metricSysCPUPercent", { desc = descS "Sys CPU %", vf = toPercent } )
        , ( "metricSysDiskUsedPercent", { desc = descKey "Disk Used %", vf = toPercent } )
        , ( "metricSysLoad", { desc = descKey "Load", vf = \p -> Round.round 2 p.value } )
        , ( "metricSysMemUsedPercent", { desc = descS "Memory used %", vf = toPercent } )
        , ( "metricSysMem", { desc = descKey "Memory", vf = toMiB } )
        , ( "metricSysNetBytesRecv", { desc = descKey "Net RX", vf = toWhole } )
        , ( "metricSysNetBytesSent", { desc = descKey "Net TX", vf = toWhole } )
        , ( "metricSysUptime", { desc = descKey "Uptime", vf = toWhole } )
        ]


toMiB : Point.Point -> String
toMiB p =
    format { usLocale | decimals = Exact 1 } (p.value / (1024 * 1024))


toPercent : Point.Point -> String
toPercent p =
    Round.round 1 p.value ++ " %"


toWhole : Point.Point -> String
toWhole p =
    format { usLocale | decimals = Exact 0 } p.value


toText : Point.Point -> String
toText p =
    if p.text == "" then
        " "

    else
        p.text


toTime : Time.Zone -> Point.Point -> String
toTime z p =
    let
        t =
            Time.millisToPosix (round p.value * 1000)
    in
    toDateTimeString z t


descS : String -> Point.Point -> String
descS d _ =
    d


descKey : String -> Point.Point -> String
descKey d p =
    d ++ " " ++ p.key
