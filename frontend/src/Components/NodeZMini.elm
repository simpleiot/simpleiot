module Components.NodeZMini exposing (view)

import Api.Node as Node
import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import List.Extra
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        serialNode =
            List.Extra.find (\n -> n.node.typ == Node.typeSerialDev) o.children

        serialNodePoints =
            case serialNode of
                Just sn ->
                    sn.node.points

                Nothing ->
                    []

        disabled =
            Point.getBool serialNodePoints Point.typeDisable ""

        connected =
            Point.getBool serialNodePoints Point.typeConnected ""

        summaryBackground =
            if disabled || not connected then
                Style.colors.ltgray

            else
                Style.colors.none
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10, Background.color summaryBackground ]
            [ Icon.serialDev
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            , viewIf (not connected) <| text "(not connected)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            180

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        optionInput =
                            NodeInputs.nodeOptionInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        numberInput =
                            NodeInputs.nodeNumberInput opts

                        simEnabled =
                            Point.getValue o.node.points "sim" ""

                        data =
                            let
                                vrmsAX =
                                    Point.getValue o.node.points "vRMS" "AX"

                                vrmsBX =
                                    Point.getValue o.node.points "vRMS" "BX"

                                vrmsOX =
                                    Point.getValue o.node.points "vRMS" "OX"

                                irmsOX =
                                    Point.getValue o.node.points "iRMS" "OX"
                            in
                            [ { typ = "vRMS", key = "AX", value = Round.round 5 vrmsAX }
                            , { typ = "vRMS", key = "BX", value = Round.round 5 vrmsBX }
                            , { typ = "vRMS", key = "OX", value = Round.round 5 vrmsOX }
                            , { typ = "iRMS", key = "OX", value = Round.round 5 irmsOX }
                            ]
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , optionInput "preferredInput"
                        "Preferred input"
                        [ ( "A", "A" )
                        , ( "B", "B" )
                        ]
                    , NodeInputs.nodeOptionInput opts
                        "comm"
                        "logLevel"
                        "Log (comm)"
                        logLevels
                    , NodeInputs.nodeOptionInput opts
                        "dsp"
                        "logLevel"
                        "Log (dsp)"
                        logLevels
                    , checkboxInput "locate" "Location mode"
                    , checkboxInput "sim"
                        "Enable simulation"
                    , viewIf (simEnabled == 1) <|
                        column [ spacing 6 ]
                            [ numberInput "X" "simVoltageA" "Sim A input voltage"
                            , numberInput "X" "simVoltageB" "Sim B input voltage"
                            , numberInput "X" "simLoadCurrent" "Sim load current"
                            , NodeInputs.nodeOptionInput opts
                                "X"
                                "simLoadProfile"
                                "Sim load profile"
                                [ ( "const", "Constant" )
                                , ( "ramp", "Ramp" )
                                , ( "random", "Random" )
                                ]
                            , NodeInputs.nodeOptionInput opts
                                "type"
                                "simEvent"
                                "Sim event type"
                                [ ( "surge", "Surge" )
                                , ( "sag", "Sag" )
                                , ( "outage", "Outage" )
                                , ( "transient", "Transient" )
                                ]
                            , NodeInputs.nodeNumberInput opts "dur" "simEvent" "Sim event dur"
                            , NodeInputs.nodeTextInput opts "per" "simEvent" "Sim event period" "0=dis, -1=oneshot, >0 = ms"
                            , NodeInputs.nodeOptionInput opts
                                "side"
                                "simEvent"
                                "Sim event side"
                                [ ( "A", "A" )
                                , ( "B", "B" )
                                ]
                            ]
                    , table
                        [ padding 7 ]
                        { data = data
                        , columns =
                            let
                                cell =
                                    el [ paddingXY 15 5, Border.width 1 ]
                            in
                            [ { header = cell <| el [ Font.bold, centerX ] <| text "Param"
                              , width = fill
                              , view = \m -> cell <| text m.typ
                              }
                            , { header = cell <| el [ Font.bold, centerX ] <| text "Loc"
                              , width = fill
                              , view = \m -> cell <| text m.key
                              }
                            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
                              , width = fill
                              , view = \m -> cell <| el [ alignRight ] <| text m.value
                              }
                            ]
                        }
                    ]

                else
                    []
               )


logLevels : List ( String, String )
logLevels =
    [ ( "", "None" )
    , ( "ERR", "Error" )
    , ( "WRN", "Warning" )
    , ( "INF", "Info" )
    , ( "DBG", "Debug" )
    ]
