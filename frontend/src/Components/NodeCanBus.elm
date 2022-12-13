module Components.NodeCanBus exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Round
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


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

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts ""

                        counterWithReset =
                            NodeInputs.nodeCounterWithReset opts ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeDevice "Device" "can0"
                    , textInput Point.typeBitRate "Bit rate" "250000"
                    , el [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "Messages in db: "
                        ++ String.fromFloat (Round.roundNum 2 (Point.getValue o.node.points Point.typeMsgsInDb ""))
                    , el [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "Signals in db: "
                        ++ String.fromFloat (Round.roundNum 2 (Point.getValue o.node.points Point.typeSignalsInDb ""))
                    , el [ width (px labelWidth) ] <|
                        el [ alignRight ] <|
                            text <|
                                "Messages in db: "
                                    ++ String.fromFloat (Round.roundNum 2 (Point.getValue o.node.points Point.typeMsgsInDb ""))
                    , el [ width (px labelWidth) ] <|
                        el [ alignRight ] <|
                            text <|
                                "Signals in db: "
                                    ++ String.fromFloat (Round.roundNum 2 (Point.getValue o.node.points Point.typeSignalsInDb ""))
                    , counterWithReset Point.typeMsgsRecvdDb Point.typeMsgsRecvdDbReset "Db msgs recieved"
                    , counterWithReset Point.typeMsgsRecvdOther Point.typeMsgsRecvdOtherReset "Other msgs recvd"
                    , checkboxInput Point.typeDisable "Disable"
                    ]

                else
                    []
               )
